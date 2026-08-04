#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webDir = path.join(root, "web");
const errors = [];

function addError(message) {
  errors.push(message);
}

function readText(file) {
  return readFileSync(file, "utf8");
}

function walk(dir, predicate = () => true) {
  const out = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = path.join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      out.push(...walk(fullPath, predicate));
      continue;
    }
    if (predicate(fullPath)) {
      out.push(fullPath);
    }
  }
  return out;
}

function rel(file) {
  return path.relative(root, file);
}

function checkJSSyntax(files) {
  for (const file of files) {
    const result = spawnSync(process.execPath, ["--check", file], {
      encoding: "utf8",
    });
    if (result.status !== 0) {
      addError(`JS syntax failed in ${rel(file)}:\n${result.stderr || result.stdout}`);
    }
  }
}

function checkJSImports(files) {
  const importPattern =
    /\bimport\s*(?:\(\s*|(?:[\s\S]*?\s+from\s*)?)["']([^"']+)["']/g;

  for (const file of files) {
    const source = readText(file);
    for (const match of source.matchAll(importPattern)) {
      const specifier = match[1];
      if (!specifier.startsWith(".")) {
        continue;
      }

      const resolved = path.resolve(path.dirname(file), specifier);
      const candidates = [resolved, `${resolved}.js`, path.join(resolved, "index.js")];
      if (!candidates.some((candidate) => existsSync(candidate))) {
        addError(`Missing JS import in ${rel(file)}: ${specifier}`);
      }
    }
  }
}

function webPathForURL(rawURL) {
  const cleanURL = rawURL.split("#", 1)[0].split("?", 1)[0].trim();
  if (
    cleanURL === "" ||
    cleanURL.startsWith("#") ||
    cleanURL.startsWith("data:") ||
    cleanURL.startsWith("mailto:") ||
    cleanURL.startsWith("tel:") ||
    /^[a-z][a-z0-9+.-]*:\/\//i.test(cleanURL)
  ) {
    return null;
  }

  if (cleanURL.startsWith("/static/")) {
    return path.join(webDir, cleanURL.slice("/static/".length));
  }
  if (cleanURL === "/manifest.webmanifest") {
    return path.join(webDir, "manifest.webmanifest");
  }
  if (cleanURL === "/sw.js") {
    return path.join(webDir, "sw.js");
  }

  return null;
}

function checkReferencedAssets() {
  const htmlPath = path.join(webDir, "index.html");
  const cssFiles = walk(path.join(webDir, "css"), (file) => file.endsWith(".css"));
  const manifestPath = path.join(webDir, "manifest.webmanifest");

  const html = readText(htmlPath);
  const attrPattern = /\b(?:href|src)=["']([^"']+)["']/g;
  for (const match of html.matchAll(attrPattern)) {
    checkURLAsset(match[1], rel(htmlPath));
  }

  const cssURLPattern = /url\(\s*["']?([^"')]+)["']?\s*\)/g;
  for (const file of cssFiles) {
    const css = readText(file);
    for (const match of css.matchAll(cssURLPattern)) {
      checkURLAsset(match[1], rel(file));
    }
  }

  const manifest = parseManifest(manifestPath);
  if (Array.isArray(manifest?.icons)) {
    for (const icon of manifest.icons) {
      if (typeof icon?.src === "string") {
        checkURLAsset(icon.src, rel(manifestPath));
      }
    }
  }
}

function checkURLAsset(url, source) {
  const file = webPathForURL(url);
  if (file && !existsSync(file)) {
    addError(`Missing asset referenced from ${source}: ${url} -> ${rel(file)}`);
  }
}

function parseManifest(manifestPath) {
  let manifest;
  try {
    manifest = JSON.parse(readText(manifestPath));
  } catch (error) {
    addError(`Invalid manifest JSON in ${rel(manifestPath)}: ${error.message}`);
    return null;
  }

  for (const field of ["name", "short_name", "start_url", "scope", "display"]) {
    if (typeof manifest[field] !== "string" || manifest[field].trim() === "") {
      addError(`Manifest field ${field} must be a non-empty string`);
    }
  }

  return manifest;
}

function checkTranslations() {
  const i18nPath = path.join(webDir, "js", "i18n.js");
  const translations = extractTranslations(i18nPath);
  if (!translations) {
    return;
  }

  const languages = Object.keys(translations);
  if (languages.length === 0) {
    addError("No translation languages found");
    return;
  }

  const allKeys = new Set(languages.flatMap((language) => Object.keys(translations[language])));
  for (const language of languages) {
    for (const key of allKeys) {
      if (!Object.hasOwn(translations[language], key)) {
        addError(`Missing ${language} translation: ${key}`);
      }
    }
  }

  for (const key of collectTranslationKeys()) {
    for (const language of languages) {
      if (!Object.hasOwn(translations[language], key)) {
        addError(`Unknown translation key in web source for ${language}: ${key}`);
      }
    }
  }
}

function extractTranslations(i18nPath) {
  const source = readText(i18nPath);
  const marker = "const translations =";
  const markerIndex = source.indexOf(marker);
  if (markerIndex < 0) {
    addError(`Cannot find translations object in ${rel(i18nPath)}`);
    return null;
  }

  const start = source.indexOf("{", markerIndex);
  if (start < 0) {
    addError(`Cannot find translations object start in ${rel(i18nPath)}`);
    return null;
  }

  const end = findMatchingBrace(source, start);
  if (end < 0) {
    addError(`Cannot find translations object end in ${rel(i18nPath)}`);
    return null;
  }

  const literal = source.slice(start, end + 1);
  try {
    return vm.runInNewContext(`(${literal})`, Object.create(null), {
      timeout: 1000,
    });
  } catch (error) {
    addError(`Cannot parse translations object in ${rel(i18nPath)}: ${error.message}`);
    return null;
  }
}

function findMatchingBrace(source, start) {
  let depth = 0;
  let quote = null;
  let escaping = false;
  let lineComment = false;
  let blockComment = false;

  for (let index = start; index < source.length; index += 1) {
    const char = source[index];
    const next = source[index + 1];

    if (lineComment) {
      if (char === "\n") {
        lineComment = false;
      }
      continue;
    }
    if (blockComment) {
      if (char === "*" && next === "/") {
        blockComment = false;
        index += 1;
      }
      continue;
    }
    if (quote) {
      if (escaping) {
        escaping = false;
      } else if (char === "\\") {
        escaping = true;
      } else if (char === quote) {
        quote = null;
      }
      continue;
    }

    if (char === "/" && next === "/") {
      lineComment = true;
      index += 1;
      continue;
    }
    if (char === "/" && next === "*") {
      blockComment = true;
      index += 1;
      continue;
    }
    if (char === "'" || char === '"' || char === "`") {
      quote = char;
      continue;
    }
    if (char === "{") {
      depth += 1;
      continue;
    }
    if (char === "}") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }

  return -1;
}

function collectTranslationKeys() {
  const keys = new Set();
  const html = readText(path.join(webDir, "index.html"));
  const dataI18nPattern = /\bdata-i18n(?:-[a-z-]+)?=["']([^"']+)["']/g;
  for (const match of html.matchAll(dataI18nPattern)) {
    keys.add(match[1]);
  }

  const jsFiles = walk(path.join(webDir, "js"), (file) => file.endsWith(".js"));
  const tCallPattern = /\bt\(\s*["']([A-Za-z0-9_.-]+)["']/g;
  for (const file of jsFiles) {
    const source = readText(file);
    for (const match of source.matchAll(tCallPattern)) {
      keys.add(match[1]);
    }
  }

  return keys;
}

const jsFiles = [
  ...walk(path.join(webDir, "js"), (file) => file.endsWith(".js")),
  path.join(webDir, "sw.js"),
];

checkJSSyntax(jsFiles);
checkJSImports(jsFiles);
checkReferencedAssets();
checkTranslations();

for (const testFile of [
  "scripts/offline-db.test.mjs",
  "scripts/network-contract.test.mjs",
  "scripts/session-cache.test.mjs",
  "scripts/session-projection.test.mjs",
  "scripts/offline-sync.test.mjs",
  "scripts/sync-status.test.mjs",
  "scripts/service-worker.test.mjs",
]) {
  const result = spawnSync(process.execPath, [testFile], { cwd: root, encoding: "utf8" });
  if (result.status !== 0) {
    addError(
      `${testFile} failed:\n${result.stderr || result.stdout || "unknown error"}`,
    );
  }
}

if (errors.length > 0) {
  console.error(errors.map((error) => `- ${error}`).join("\n"));
  process.exit(1);
}

console.log(`web checks passed (${jsFiles.length} JS files)`);
