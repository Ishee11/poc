function clone(value) {
  return value === undefined ? undefined : structuredClone(value);
}

function nameList(values) {
  const list = [...values];
  list.contains = (value) => list.includes(value);
  return list;
}

function keyFor(record, keyPath) {
  if (Array.isArray(keyPath)) return keyPath.map((part) => record[part]);
  return record[keyPath];
}

function serializeKey(key) {
  return JSON.stringify(key);
}

function compareKeys(left, right) {
  const leftParts = Array.isArray(left) ? left : [left];
  const rightParts = Array.isArray(right) ? right : [right];
  for (let index = 0; index < Math.max(leftParts.length, rightParts.length); index += 1) {
    if (leftParts[index] === rightParts[index]) continue;
    return leftParts[index] < rightParts[index] ? -1 : 1;
  }
  return 0;
}

function cloneStore(store) {
  return {
    keyPath: clone(store.keyPath),
    indexes: new Map(
      [...store.indexes].map(([name, definition]) => [name, clone(definition)]),
    ),
    records: new Map(
      [...store.records].map(([key, entry]) => [key, { key: clone(entry.key), value: clone(entry.value) }]),
    ),
  };
}

class FakeRequest {
  result = undefined;
  error = null;
  onsuccess = null;
  onerror = null;
}

class FakeTransaction {
  constructor(factory, databaseState, storeNames, mode, { upgrade = false } = {}) {
    this.factory = factory;
    this.databaseState = databaseState;
    this.storeNames = new Set(storeNames);
    this.mode = mode;
    this.upgrade = upgrade;
    this.error = null;
    this.oncomplete = null;
    this.onabort = null;
    this.onerror = null;
    this.pending = 0;
    this.finished = false;
    this.aborted = false;
    this.stores = new Map();
    for (const storeName of this.storeNames) {
      const store = databaseState.stores.get(storeName);
      if (store) this.stores.set(storeName, cloneStore(store));
    }
    queueMicrotask(() => this.maybeComplete());
  }

  objectStore(name) {
    if (!this.storeNames.has(name) || !this.stores.has(name)) {
      throw new Error(`Object store ${name} is not part of this transaction`);
    }
    return new FakeObjectStore(this, name);
  }

  addCreatedStore(name, store) {
    this.storeNames.add(name);
    this.stores.set(name, store);
  }

  request(work, { writeStore = "" } = {}) {
    const request = new FakeRequest();
    this.pending += 1;
    queueMicrotask(() => {
      if (this.aborted) return;
      try {
        if (writeStore && this.factory.consumePutFailure(writeStore)) {
          throw new Error(`Injected ${writeStore} put failure`);
        }
        request.result = work();
        request.onsuccess?.({ target: request });
      } catch (error) {
        request.error = error;
        request.onerror?.({ target: request });
        this.fail(error);
      } finally {
        this.pending -= 1;
        queueMicrotask(() => this.maybeComplete());
      }
    });
    return request;
  }

  fail(error) {
    if (this.finished || this.aborted) return;
    this.error = error;
    this.onerror?.({ target: this });
    this.abort();
  }

  abort() {
    if (this.finished || this.aborted) return;
    this.aborted = true;
    queueMicrotask(() => this.onabort?.({ target: this }));
  }

  maybeComplete() {
    if (this.finished || this.aborted || this.pending > 0) return;
    this.finished = true;
    if (this.mode === "readwrite" || this.upgrade) {
      for (const [name, store] of this.stores) {
        this.databaseState.stores.set(name, cloneStore(store));
      }
    }
    this.oncomplete?.({ target: this });
  }
}

class FakeObjectStore {
  constructor(transaction, name) {
    this.transaction = transaction;
    this.name = name;
  }

  get state() {
    return this.transaction.stores.get(this.name);
  }

  get indexNames() {
    return nameList(this.state.indexes.keys());
  }

  createIndex(name, keyPath, options = {}) {
    if (this.state.indexes.has(name)) throw new Error(`Index ${name} already exists`);
    this.state.indexes.set(name, { keyPath: clone(keyPath), unique: Boolean(options.unique) });
    return new FakeIndex(this.transaction, this.name, name);
  }

  index(name) {
    if (!this.state.indexes.has(name)) throw new Error(`Index ${name} does not exist`);
    return new FakeIndex(this.transaction, this.name, name);
  }

  put(value) {
    return this.transaction.request(
      () => {
        if (this.transaction.mode !== "readwrite" && !this.transaction.upgrade) {
          throw new Error("Transaction is readonly");
        }
        const key = keyFor(value, this.state.keyPath);
        if (key === undefined || key === null || key === "") {
          throw new Error(`Missing key ${this.state.keyPath}`);
        }
        this.state.records.set(serializeKey(key), { key: clone(key), value: clone(value) });
        return clone(key);
      },
      { writeStore: this.name },
    );
  }

  get(key) {
    return this.transaction.request(() => clone(this.state.records.get(serializeKey(key))?.value));
  }

  getAll() {
    return this.transaction.request(() =>
      [...this.state.records.values()].map((entry) => clone(entry.value)),
    );
  }

  delete(key) {
    return this.transaction.request(
      () => {
        this.state.records.delete(serializeKey(key));
        return undefined;
      },
      { writeStore: this.name },
    );
  }
}

class FakeIndex {
  constructor(transaction, storeName, indexName) {
    this.transaction = transaction;
    this.storeName = storeName;
    this.indexName = indexName;
  }

  getAll() {
    return this.transaction.request(() => {
      const store = this.transaction.stores.get(this.storeName);
      const definition = store.indexes.get(this.indexName);
      return [...store.records.values()]
        .sort((left, right) =>
          compareKeys(
            keyFor(left.value, definition.keyPath),
            keyFor(right.value, definition.keyPath),
          ),
        )
        .map((entry) => clone(entry.value));
    });
  }
}

class FakeDatabase {
  constructor(factory, state) {
    this.factory = factory;
    this.state = state;
    this.onversionchange = null;
  }

  get version() {
    return this.state.version;
  }

  get objectStoreNames() {
    return nameList(this.state.stores.keys());
  }

  createObjectStore(name, { keyPath }) {
    if (!this.upgradeTransaction) throw new Error("No upgrade transaction");
    if (this.state.stores.has(name) || this.upgradeTransaction.stores.has(name)) {
      throw new Error(`Object store ${name} already exists`);
    }
    const store = { keyPath, indexes: new Map(), records: new Map() };
    this.upgradeTransaction.addCreatedStore(name, store);
    return new FakeObjectStore(this.upgradeTransaction, name);
  }

  transaction(storeNames, mode = "readonly") {
    const names = Array.isArray(storeNames) ? storeNames : [storeNames];
    for (const name of names) {
      if (!this.state.stores.has(name)) throw new Error(`Unknown object store ${name}`);
    }
    return new FakeTransaction(this.factory, this.state, names, mode);
  }

  close() {}
}

export class FakeIndexedDBFactory {
  constructor() {
    this.databases = new Map();
    this.putFailures = [];
  }

  failNextPut(storeName) {
    this.putFailures.push(storeName);
  }

  consumePutFailure(storeName) {
    const index = this.putFailures.indexOf(storeName);
    if (index < 0) return false;
    this.putFailures.splice(index, 1);
    return true;
  }

  seed(databaseName, storeName, value) {
    const database = this.databases.get(databaseName);
    const store = database?.stores.get(storeName);
    if (!store) throw new Error(`Unknown object store ${storeName}`);
    const key = keyFor(value, store.keyPath);
    store.records.set(serializeKey(key), { key: clone(key), value: clone(value) });
  }

  hasIndex(databaseName, storeName, indexName) {
    return Boolean(
      this.databases.get(databaseName)?.stores.get(storeName)?.indexes.has(indexName),
    );
  }

  open(name, version) {
    const request = new FakeRequest();
    request.onupgradeneeded = null;
    request.onblocked = null;
    request.transaction = null;

    queueMicrotask(() => {
      const existing = this.databases.get(name);
      const requestedVersion = version ?? existing?.version ?? 1;
      if (existing && requestedVersion < existing.version) {
        request.error = new Error("VersionError");
        request.onerror?.({ target: request });
        return;
      }

      const state = existing || { version: 0, stores: new Map() };
      const database = new FakeDatabase(this, state);
      request.result = database;

      if (requestedVersion > state.version) {
        const oldVersion = state.version;
        const transaction = new FakeTransaction(
          this,
          state,
          state.stores.keys(),
          "readwrite",
          { upgrade: true },
        );
        database.upgradeTransaction = transaction;
        request.transaction = transaction;
        request.onupgradeneeded?.({ oldVersion, newVersion: requestedVersion, target: request });
        transaction.oncomplete = () => {
          state.version = requestedVersion;
          this.databases.set(name, state);
          database.upgradeTransaction = null;
          request.onsuccess?.({ target: request });
        };
        transaction.onabort = () => {
          request.error = transaction.error || new Error("Upgrade aborted");
          request.onerror?.({ target: request });
        };
        transaction.maybeComplete();
        return;
      }

      request.onsuccess?.({ target: request });
    });
    return request;
  }
}
