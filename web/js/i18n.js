const STORAGE_KEY = "poker-ui-language";
const DEFAULT_LANGUAGE = "en";
const SUPPORTED_LANGUAGES = new Set(["en", "ru"]);

let currentLanguage = detectLanguage();
const listeners = new Set();

const translations = {
  en: {
    "app.title": "Poker Session Control",
    "app.subtitle": "Control panel for poker cash games.",
    "nav.page": "Page navigation",
    "nav.backHome": "Back to Home",
    "nav.backSession": "Back to Session",
    "lobby.connectTitle": "Connect to Active Session",
    "lobby.latestActiveSession": "Latest active session",
    "lobby.openWorkspace": "Open Workspace",
    "lobby.startTitle": "Start a New Session",
    "lobby.chipRate": "Rate (1 ruble for N chips)",
    "lobby.startHint":
      "New sessions are created as active and become visible in the list below.",
    "lobby.startSession": "Start Session",
    "lobby.addPlayerTitle": "Add New Player",
    "lobby.playerName": "Player Name",
    "lobby.playerNamePlaceholder": "Enter player name",
    "lobby.createPlayer": "Create Player",
    "lobby.sessions": "Sessions",
    "lobby.players": "Players",
    "language.title": "Interface Language",
    "language.hint":
      "Russian is selected automatically for Russian-language devices. Everyone else gets English.",
    "debug.deletePlayer": "Delete Player",
    "debug.deleteSession": "Delete Session",
    "session.title": "Session",
    "session.finish": "Finish Session",
    "session.chipRate": "Rate",
    "session.chipRateValue": "1 ₽ = {chips} chips",
    "session.totalBuyIn": "Total Buy In",
    "session.totalCashOut": "Total Cash Out",
    "session.onTable": "On Table",
    "session.totalMoneyIn": "Total Money In",
    "session.addExistingPlayer": "Add Player + Buy In",
    "session.createNewPlayer": "Create New Player",
    "session.players": "Players",
    "session.actions": "Actions",
    "session.actionsHint":
      "You can choose only players already added to this session.",
    "session.player": "Player",
    "session.selectPlayer": "Select player",
    "session.chips": "Chips",
    "session.buyIn": "Rebuy",
    "session.cashOut": "Cash Out",
    "session.latestOperations": "Latest Operations",
    "player.title": "Player",
    "player.dataPlaceholder": "Player data will appear here",
    "player.from": "From",
    "player.to": "To",
    "player.applyPeriod": "Apply Period",
    "player.allTime": "All Time",
    "player.sessions": "Sessions",
    "player.totalBuyIn": "Total Buy In",
    "player.totalCashOut": "Total Cash Out",
    "player.profitMoney": "Profit Money",
    "table.session": "Session",
    "table.status": "Status",
    "table.buyIn": "Buy In",
    "table.cashOut": "Cash Out",
    "table.profitChips": "Profit Chips",
    "table.profit": "Profit",
    "table.lastActivity": "Last Activity",
    "common.open": "Open",
    "common.reverse": "Reverse",
    "common.cancel": "Cancel",
    "common.confirm": "Confirm",
    "common.noSessions": "No sessions",
    "common.noPlayers": "No players",
    "common.noOperations": "No operations yet",
    "common.noData": "No data",
    "common.status": "Status",
    "common.players": "Players",
    "common.sessions": "Sessions",
    "common.profit": "Profit",
    "common.buyIn": "Buy in",
    "common.cashOut": "Cash out",
    "common.inGame": "In game",
    "common.settled": "Settled",
    "common.lastActivity": "Last activity",
    "status.active": "Active",
    "status.finished": "Finished",
    "operation.buy_in": "Buy in",
    "operation.cash_out": "Cash out",
    "operation.reversal": "Reversal",
    "notice.noSession": "No session available to open.",
    "notice.validChipRate": "Enter a valid rate.",
    "notice.sessionStarted": "Session started.",
    "notice.enterPlayerName": "Enter player name.",
    "notice.playerCreated": "Player {name} created.",
    "notice.selectPlayerAndChips":
      "Select a player and enter a valid chip amount.",
    "notice.buyInRecorded": "Buy in recorded for {name}.",
    "notice.cashOutRecorded": "Cash out recorded for {name}.",
    "notice.noAvailablePlayers":
      "No available players to add. Create a new player instead.",
    "notice.choosePlayerAndBuyIn":
      "Choose a player and enter a valid initial buy in.",
    "notice.playerAdded": "Player {name} added to session.",
    "notice.enterPlayerAndBuyIn":
      "Enter player name and a valid initial buy in.",
    "notice.playerCreatedAndAdded": "Player {name} created and added to session.",
    "notice.cannotFinish":
      "Cannot finish session yet. Remaining chips on table: {chips}.",
    "notice.sessionFinished": "Session finished.",
    "notice.operationReversed": "Operation reversed.",
    "hint.finishBlocked":
      "Cannot finish session yet: cash out or reverse operations until ON TABLE becomes 0.",
    "modal.startTitle": "Start Session",
    "modal.startDescription": "Start a new session with rate 1 ₽ = {chipRate} chips?",
    "modal.createPlayerTitle": "Create Player",
    "modal.createPlayerDescription": "Create player \"{name}\"?",
    "modal.confirmBuyInTitle": "Confirm Buy In",
    "modal.confirmBuyInDescription": "Add {chips} chips for {name}?",
    "modal.confirmCashOutTitle": "Confirm Cash Out",
    "modal.confirmCashOutDescription": "Cash out {chips} chips for {name}?",
    "modal.addPlayerTitle": "Add Player to Session",
    "modal.addPlayerDescription":
      "The player will appear in the session after the first buy in. This matches the current backend flow.",
    "modal.addToSession": "Add to Session",
    "modal.initialBuyIn": "Initial Buy In",
    "modal.createNewPlayerTitle": "Create New Player",
    "modal.createNewPlayerDescription":
      "A new player is created in the global player list and immediately added to this session through the first buy in.",
    "modal.createAndAdd": "Create and Add",
    "modal.finishTitle": "Finish Session",
    "modal.finishDescription":
      "Finish the session now? The backend allows this only when total buy in equals total cash out.",
    "modal.reverseTitle": "Reverse Operation",
    "modal.reverseDescription": "Reverse {type} for {name} with {chips} chips?",
    "modal.deletePlayerTitle": "Delete Player",
    "modal.deletePlayerDescription":
      "Delete player {name}? This also deletes this player's operations and recalculates affected sessions.",
    "modal.deleteSessionTitle": "Delete Session",
    "modal.deleteSessionDescription":
      "Delete this session and all its operations? This cannot be undone.",
    "error.fallback": "Request failed",
    "error.failedStartSession": "Failed to start session",
    "error.failedCreatePlayer": "Failed to create player",
    "error.failedLoadSession": "Failed to load session",
    "error.failedBuyIn": "Failed to apply buy in",
    "error.failedCashOut": "Failed to apply cash out",
    "error.failedAddPlayer": "Failed to add player to session",
    "error.failedCreateAdd": "Player created, but add to session failed",
    "error.failedFinish": "Failed to finish session",
    "error.failedReverse": "Failed to reverse operation",
    "error.failedRefresh": "Failed to refresh session",
    "error.failedDeletePlayer": "Failed to delete player",
    "error.failedDeleteSession": "Failed to delete session",
    "notice.playerDeleted": "Player deleted.",
    "notice.sessionDeleted": "Session deleted.",
    "error.sessionNotBalanced":
      "Session is not balanced yet. Remaining chips on table: {chips}.",
    "error.invalid_request_id": "Request ID is missing.",
    "error.invalid_chips": "Enter a chip amount greater than zero.",
    "error.invalid_cash_out": "Cash out is not valid for the current table state.",
    "error.invalid_operation": "This operation cannot be performed.",
    "error.player_not_found": "Selected player does not exist.",
    "error.session_not_found": "Session was not found.",
    "error.session_not_active": "Session is not active anymore.",
    "error.session_finished": "Session is already finished.",
    "error.operation_not_found": "Operation was not found.",
    "error.operation_already_reversed": "Operation has already been reversed.",
    "error.internal_error": "Server returned an internal error.",
  },
  ru: {
    "app.title": "Управление покерной сессией",
    "app.subtitle": "Панель управления покерными кэш-играми.",
    "nav.page": "Навигация по странице",
    "nav.backHome": "На главную",
    "nav.backSession": "К сессии",
    "lobby.connectTitle": "Открыть активную сессию",
    "lobby.latestActiveSession": "Последняя активная сессия",
    "lobby.openWorkspace": "Открыть сессию",
    "lobby.startTitle": "Начать новую сессию",
    "lobby.chipRate": "Курс (1 рубль за N фишек)",
    "lobby.startHint":
      "Новые сессии создаются активными и появляются в списке ниже.",
    "lobby.startSession": "Начать сессию",
    "lobby.addPlayerTitle": "Добавить нового игрока",
    "lobby.playerName": "Имя игрока",
    "lobby.playerNamePlaceholder": "Введите имя игрока",
    "lobby.createPlayer": "Создать игрока",
    "lobby.sessions": "Сессии",
    "lobby.players": "Игроки",
    "language.title": "Язык интерфейса",
    "language.hint":
      "Русский выбирается автоматически для русскоязычных устройств. Для остальных используется английский.",
    "debug.deletePlayer": "Удалить игрока",
    "debug.deleteSession": "Удалить сессию",
    "session.title": "Сессия",
    "session.finish": "Завершить сессию",
    "session.chipRate": "Курс",
    "session.chipRateValue": "1 ₽ = {chips} фишек",
    "session.totalBuyIn": "Всего внесено",
    "session.totalCashOut": "Всего выведено",
    "session.onTable": "На столе",
    "session.totalMoneyIn": "Всего внесено деньгами",
    "session.addExistingPlayer": "Добавить игрока + бай-ин",
    "session.createNewPlayer": "Создать нового игрока",
    "session.players": "Игроки",
    "session.actions": "Действия",
    "session.actionsHint":
      "Выбрать можно только игроков, которые уже добавлены в эту сессию.",
    "session.player": "Игрок",
    "session.selectPlayer": "Выберите игрока",
    "session.chips": "Фишки",
    "session.buyIn": "Ребай",
    "session.cashOut": "Кэш-аут",
    "session.latestOperations": "Последние операции",
    "player.title": "Игрок",
    "player.dataPlaceholder": "Данные игрока появятся здесь",
    "player.from": "С",
    "player.to": "По",
    "player.applyPeriod": "Применить период",
    "player.allTime": "За все время",
    "player.sessions": "Сессии",
    "player.totalBuyIn": "Всего внесено",
    "player.totalCashOut": "Всего выведено",
    "player.profitMoney": "Профит деньгами",
    "table.session": "Сессия",
    "table.status": "Статус",
    "table.buyIn": "Бай-ин",
    "table.cashOut": "Кэш-аут",
    "table.profitChips": "Профит фишками",
    "table.profit": "Профит",
    "table.lastActivity": "Последняя активность",
    "common.open": "Открыть",
    "common.reverse": "Отменить",
    "common.cancel": "Отмена",
    "common.confirm": "Подтвердить",
    "common.noSessions": "Сессий нет",
    "common.noPlayers": "Игроков нет",
    "common.noOperations": "Операций пока нет",
    "common.noData": "Нет данных",
    "common.status": "Статус",
    "common.players": "Игроки",
    "common.sessions": "Сессии",
    "common.profit": "Профит",
    "common.buyIn": "Бай-ин",
    "common.cashOut": "Кэш-аут",
    "common.inGame": "В игре",
    "common.settled": "Рассчитан",
    "common.lastActivity": "Последняя активность",
    "status.active": "Активна",
    "status.finished": "Завершена",
    "operation.buy_in": "Бай-ин",
    "operation.cash_out": "Кэш-аут",
    "operation.reversal": "Отмена",
    "notice.noSession": "Нет сессии для открытия.",
    "notice.validChipRate": "Введите корректный курс.",
    "notice.sessionStarted": "Сессия начата.",
    "notice.enterPlayerName": "Введите имя игрока.",
    "notice.playerCreated": "Игрок {name} создан.",
    "notice.selectPlayerAndChips": "Выберите игрока и введите количество фишек.",
    "notice.buyInRecorded": "Бай-ин записан для {name}.",
    "notice.cashOutRecorded": "Кэш-аут записан для {name}.",
    "notice.noAvailablePlayers":
      "Нет доступных игроков для добавления. Создайте нового игрока.",
    "notice.choosePlayerAndBuyIn":
      "Выберите игрока и введите корректный начальный бай-ин.",
    "notice.playerAdded": "Игрок {name} добавлен в сессию.",
    "notice.enterPlayerAndBuyIn":
      "Введите имя игрока и корректный начальный бай-ин.",
    "notice.playerCreatedAndAdded": "Игрок {name} создан и добавлен в сессию.",
    "notice.cannotFinish":
      "Сессию пока нельзя завершить. Осталось фишек на столе: {chips}.",
    "notice.sessionFinished": "Сессия завершена.",
    "notice.operationReversed": "Операция отменена.",
    "hint.finishBlocked":
      "Сессию пока нельзя завершить: сделайте кэш-аут или отмените операции, пока НА СТОЛЕ не станет 0.",
    "modal.startTitle": "Начать сессию",
    "modal.startDescription": "Начать новую сессию с курсом 1 ₽ = {chipRate} фишек?",
    "modal.createPlayerTitle": "Создать игрока",
    "modal.createPlayerDescription": "Создать игрока «{name}»?",
    "modal.confirmBuyInTitle": "Подтвердить бай-ин",
    "modal.confirmBuyInDescription": "Добавить {chips} фишек для {name}?",
    "modal.confirmCashOutTitle": "Подтвердить кэш-аут",
    "modal.confirmCashOutDescription": "Вывести {chips} фишек для {name}?",
    "modal.addPlayerTitle": "Добавить игрока в сессию",
    "modal.addPlayerDescription":
      "Игрок появится в сессии после первого бай-ина. Это соответствует текущей логике бэкенда.",
    "modal.addToSession": "Добавить в сессию",
    "modal.initialBuyIn": "Начальный бай-ин",
    "modal.createNewPlayerTitle": "Создать нового игрока",
    "modal.createNewPlayerDescription":
      "Новый игрок будет создан в общем списке и сразу добавлен в эту сессию через первый бай-ин.",
    "modal.createAndAdd": "Создать и добавить",
    "modal.finishTitle": "Завершить сессию",
    "modal.finishDescription":
      "Завершить сессию сейчас? Бэкенд разрешает это только когда общий бай-ин равен общему кэш-ауту.",
    "modal.reverseTitle": "Отменить операцию",
    "modal.reverseDescription": "Отменить {type} для {name} на {chips} фишек?",
    "modal.deletePlayerTitle": "Удалить игрока",
    "modal.deletePlayerDescription":
      "Удалить игрока {name}? Это также удалит операции этого игрока и пересчитает затронутые сессии.",
    "modal.deleteSessionTitle": "Удалить сессию",
    "modal.deleteSessionDescription":
      "Удалить эту сессию и все ее операции? Это действие нельзя отменить.",
    "error.fallback": "Запрос не выполнен",
    "error.failedStartSession": "Не удалось начать сессию",
    "error.failedCreatePlayer": "Не удалось создать игрока",
    "error.failedLoadSession": "Не удалось загрузить сессию",
    "error.failedBuyIn": "Не удалось применить бай-ин",
    "error.failedCashOut": "Не удалось применить кэш-аут",
    "error.failedAddPlayer": "Не удалось добавить игрока в сессию",
    "error.failedCreateAdd": "Игрок создан, но добавить его в сессию не удалось",
    "error.failedFinish": "Не удалось завершить сессию",
    "error.failedReverse": "Не удалось отменить операцию",
    "error.failedRefresh": "Не удалось обновить сессию",
    "error.failedDeletePlayer": "Не удалось удалить игрока",
    "error.failedDeleteSession": "Не удалось удалить сессию",
    "notice.playerDeleted": "Игрок удален.",
    "notice.sessionDeleted": "Сессия удалена.",
    "error.sessionNotBalanced":
      "Сессия пока не сбалансирована. Осталось фишек на столе: {chips}.",
    "error.invalid_request_id": "Отсутствует ID запроса.",
    "error.invalid_chips": "Введите количество фишек больше нуля.",
    "error.invalid_cash_out": "Кэш-аут недопустим для текущего состояния стола.",
    "error.invalid_operation": "Эту операцию нельзя выполнить.",
    "error.player_not_found": "Выбранный игрок не найден.",
    "error.session_not_found": "Сессия не найдена.",
    "error.session_not_active": "Сессия больше не активна.",
    "error.session_finished": "Сессия уже завершена.",
    "error.operation_not_found": "Операция не найдена.",
    "error.operation_already_reversed": "Операция уже отменена.",
    "error.internal_error": "Сервер вернул внутреннюю ошибку.",
  },
};

export function initI18n() {
  document.documentElement.lang = currentLanguage;
  applyTranslations();
}

export function getLanguage() {
  return currentLanguage;
}

export function setLanguage(language) {
  if (!SUPPORTED_LANGUAGES.has(language) || language === currentLanguage) {
    return;
  }

  currentLanguage = language;
  saveLanguage(language);
  document.documentElement.lang = language;
  applyTranslations();
  listeners.forEach((listener) => listener(language));
}

export function onLanguageChange(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function t(key, params = {}) {
  const template =
    translations[currentLanguage]?.[key] ?? translations[DEFAULT_LANGUAGE][key] ?? key;

  return Object.entries(params).reduce(
    (result, [name, value]) => result.replaceAll(`{${name}}`, String(value)),
    template,
  );
}

export function statusLabel(status) {
  const key = `status.${status}`;
  const label = t(key);
  return label === key ? status : label;
}

export function operationLabel(type) {
  const key = `operation.${type}`;
  const label = t(key);
  return label === key ? type : label;
}

export function applyTranslations(root = document) {
  document.title = t("app.title");

  root.querySelectorAll("[data-i18n]").forEach((element) => {
    element.textContent = t(element.dataset.i18n);
  });

  root.querySelectorAll("[data-i18n-placeholder]").forEach((element) => {
    element.setAttribute("placeholder", t(element.dataset.i18nPlaceholder));
  });

  root.querySelectorAll("[data-i18n-aria-label]").forEach((element) => {
    element.setAttribute("aria-label", t(element.dataset.i18nAriaLabel));
  });

  const languageSelect = document.getElementById("language-select");
  if (languageSelect) {
    languageSelect.value = currentLanguage;
  }
}

function detectLanguage() {
  const saved = loadSavedLanguage();
  if (SUPPORTED_LANGUAGES.has(saved)) {
    return saved;
  }

  const languages = navigator.languages?.length
    ? navigator.languages
    : [navigator.language || DEFAULT_LANGUAGE];

  return languages.some((language) => language.toLowerCase().startsWith("ru"))
    ? "ru"
    : DEFAULT_LANGUAGE;
}

function loadSavedLanguage() {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function saveLanguage(language) {
  try {
    localStorage.setItem(STORAGE_KEY, language);
  } catch {}
}
