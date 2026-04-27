const STORAGE_KEY = "poker-ui-language";
const DEFAULT_LANGUAGE = "en";
const SUPPORTED_LANGUAGES = new Set(["en", "ru"]);

let currentLanguage = detectLanguage();
const listeners = new Set();

const translations = {
  en: {
    "app.title": "Poker Session Control",
    "app.subtitle": "Control panel for cash games.",
    "nav.page": "Page navigation",
    "nav.backHome": "Back to Home",
    "nav.backSession": "Back to Session",
    "auth.label": "Authentication",
    "auth.email": "Email",
    "auth.password": "Password",
    "auth.login": "Login",
    "auth.logout": "Logout",
    "auth.signedIn": "Signed in",
    "auth.noAccount": "No account yet?",
    "auth.register": "Register",
    "guest.player": "Guest player",
    "guest.noPlayer": "No player selected",
    "account.label": "Account",
    "account.title": "Account",
    "account.hint": "Link any free player.",
    "account.loading": "Loading account...",
    "account.loginRequired": "Log in to manage linked players.",
    "account.noLinkedPlayers": "No linked players yet.",
    "account.noAvailablePlayers": "No free players to link.",
    "account.selectPlayer": "Select player",
    "account.linkPlayer": "Link",
    "account.unlinkPlayer": "Unlink",
    "lobby.connectTitle": "Connect to Active Session",
    "lobby.latestActiveSession": "Latest active session",
    "lobby.openWorkspace": "Open Workspace",
    "lobby.openBlindsClock": "Blind Timer",
    "lobby.startTitle": "Start a New Session",
    "lobby.chipRate": "Rate (1 {currencySymbol} for N chips)",
    "lobby.currency": "Currency",
    "lobby.bigBlind": "BB (big blind size)",
    "lobby.startSession": "Start Session",
    "lobby.addPlayerTitle": "Add New Player",
    "lobby.playerName": "Player Name",
    "lobby.playerNamePlaceholder": "Enter player name",
    "lobby.createPlayer": "Create Player",
    "lobby.sessions": "Sessions",
    "lobby.players": "Players",
    "lobby.playersSort": "Sort",
    "language.title": "Interface Language",
    "language.hint":
      "Russian is selected automatically for Russian-language devices. Everyone else gets English.",
    "debug.deletePlayer": "Delete Player",
    "debug.deleteSession": "Delete Session",
    "debug.deleteFinish": "Cancel Finish",
    "debug.renamePlayer": "Rename Player",
    "debug.editSessionConfig": "Edit rate and big blind",
    "session.title": "Session",
    "session.finish": "Finish Session",
    "session.chipRate": "Rate",
    "session.chipRateValue": "1 {currencySymbol} = {chips} chips",
    "session.bigBlind": "BB",
    "session.bigBlindShort": "BB",
    "session.totalBuyIn": "Buy In",
    "session.totalCashOut": "Cash Out",
    "session.onTable": "Chips on Table",
    "session.totalMoneyIn": "Total Money In",
    "session.addExistingPlayer": "Add Player to Session",
    "session.createNewPlayer": "Create New Player",
    "session.players": "Players",
    "session.actions": "Actions",
    "session.actionsHint": "Cash-out is available only for players currently in game.",
    "session.buyInHint":
      "Add an existing player through Buy-in or create a new player here.",
    "session.cashOutHint":
      "After cash-out, the player is considered settled. Make a rebuy to continue.",
    "session.player": "Player",
    "session.selectPlayer": "Select player",
    "session.chips": "Chips",
    "session.buyIn": "Rebuy",
    "session.cashOut": "Cash-out",
    "session.latestOperations": "Latest Operations",
    "blinds.title": "Blind Timer",
    "blinds.description":
      "Separate screen for tournament blinds with pause, resume, and editable future levels.",
    "blinds.currentLevel": "Current Level",
    "blinds.currentBlinds": "Current Blinds",
    "blinds.nextLevel": "Next Level",
    "blinds.timeLeft": "Time Left",
    "blinds.controls": "Controls",
    "blinds.totalLevels": "Total Levels",
    "blinds.upcomingLevels": "Upcoming",
    "blinds.structure": "Blind Structure",
    "blinds.selectedLevel": "Selected Level",
    "blinds.addLevel": "Add Level",
    "blinds.deleteAllLevels": "Delete All Levels",
    "blinds.smallBlind": "SB",
    "blinds.bigBlind": "BB",
    "blinds.duration": "Minutes",
    "blinds.deleteLevel": "Delete",
    "blinds.editHint":
      "Completed levels are locked. Pause the timer to edit current or future levels.",
    "blinds.maintenance": "Maintenance",
    "blinds.resetHint": "Reset returns the timer to the first level and stops the clock.",
    "blinds.maintenanceHint":
      "Dangerous actions are separated from the live timer so they are harder to hit by mistake.",
    "blinds.toolStructure": "Structure",
    "blinds.toolClock": "Clock",
    "blinds.toolDanger": "Danger Zone",
    "blinds.start": "Start",
    "blinds.pause": "Pause",
    "blinds.resume": "Resume",
    "blinds.reset": "Reset",
    "blinds.noNextLevel": "No next level",
    "blinds.noLevels": "No levels yet",
    "blinds.levelCurrent": "current",
    "blinds.levelLocked": "locked",
    "blinds.lockedWhileRunning": "Pause the timer to edit levels.",
    "blinds.lockedCompletedLevel": "Completed levels cannot be edited.",
    "blinds.statusStopped": "Stopped",
    "blinds.statusRunning": "Running",
    "blinds.statusPaused": "Paused",
    "blinds.statusFinished": "Finished",
    "blinds.levelValue": "Level {level}",
    "blinds.resetTitle": "Reset Timer",
    "blinds.resetDescription": "Reset the clock to the first level and stop it?",
    "blinds.deleteLevelTitle": "Delete Level",
    "blinds.deleteLevelDescription": "Delete level {level} from the structure?",
    "blinds.deleteAllLevelsTitle": "Delete All Levels",
    "blinds.deleteAllLevelsDescription": "Delete the entire blind structure?",
    "player.title": "Player",
    "player.dataPlaceholder": "Player data will appear here",
    "player.from": "From",
    "player.to": "To",
    "player.applyPeriod": "Apply Period",
    "player.allTime": "All Time",
    "player.period": "Period",
    "player.selectPeriod": "Select Period",
    "player.sessions": "Sessions",
    "player.totalBuyIn": "Total Buy-in Chips",
    "player.totalCashOut": "Total Cash-out Chips",
    "player.totalBuyInMoney": "Total Buy-in Money",
    "player.totalCashOutMoney": "Total Cash-out Money",
    "player.pnl": "PnL",
    "player.avgProfitPerSession": "Avg Profit",
    "player.roi": "ROI",
    "player.avgBuyInPerSession": "Avg Buy-in",
    "player.profitMoney": "Profit Money",
    "player.hint.sessions": "Number of sessions in the selected period.",
    "player.hint.totalBuyIn": "Total chips bought in by this player.",
    "player.hint.totalCashOut": "Total chips cashed out by this player.",
    "player.hint.totalBuyInMoney": "Total buy-in converted to money by each session rate.",
    "player.hint.totalCashOutMoney": "Total cash-out converted to money by each session rate.",
    "player.hint.pnl": "PnL = Cash-out amount - Buy-in amount.",
    "player.hint.avgProfitPerSession": "Average profit per session = Total PnL / Sessions count.",
    "player.hint.roi": "ROI = Total PnL / Total buy-in amount * 100.",
    "player.hint.avgBuyInPerSession": "Average buy-in per session in chips = Total buy-in chips / Sessions count.",
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
    "common.close": "Close",
    "common.save": "Save",
    "common.noSessions": "No sessions",
    "common.noPlayers": "No players",
    "common.noOperations": "No operations yet",
    "common.noData": "No data",
    "common.status": "Status",
    "common.players": "Players",
    "common.sessions": "Sessions",
    "common.profit": "Profit",
    "common.buyIn": "Buy in",
    "common.totalBuyIn": "Total Buy-in",
    "common.chips": "chips",
    "common.cashOut": "Cash out",
    "common.inGame": "In game",
    "common.settled": "Settled",
    "common.lastActivity": "Last activity",
    "sort.lastActivity": "Last activity",
    "sort.sessionsCount": "Sessions count",
    "sort.profit": "Profit",
    "sort.name": "Name",
    "status.active": "Active",
    "status.finished": "Finished",
    "operation.buy_in": "Buy in",
    "operation.cash_out": "Cash out",
    "operation.reversal": "Reversal",
    "operation.finish": "Finish Session",
    "notice.noSession": "No session available to open.",
    "notice.validChipRate": "Enter a valid rate.",
    "notice.validBigBlind": "Enter a valid big blind.",
    "notice.sessionStarted": "Session started.",
    "notice.authCredentialsRequired": "Enter email and password.",
    "notice.loginSuccess": "Login successful.",
    "notice.logoutSuccess": "Logged out.",
    "notice.registrationPending": "Registration is not connected yet.",
    "notice.registrationSuccess": "Registration complete.",
    "notice.selectAccountPlayer": "Select a player to link.",
    "notice.accountPlayerLinked": "Player linked to your account.",
    "notice.accountPlayerUnlinked": "Player unlinked from your account.",
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
    "notice.playerAlreadyCashedOut":
      "This player already cashed out. Add them again through Buy-in if needed.",
    "notice.enterPlayerAndBuyIn":
      "Enter player name and a valid initial buy in.",
    "notice.playerCreatedAndAdded": "Player {name} created and added to session.",
    "notice.cannotFinish":
      "Cannot finish session yet. Remaining chips on table: {chips}.",
    "notice.sessionFinished": "Session finished.",
    "notice.operationReversed": "Operation reversed.",
    "notice.blindsStarted": "Blind timer started.",
    "notice.blindsPaused": "Blind timer paused.",
    "notice.blindsResumed": "Blind timer resumed.",
    "notice.blindsReset": "Blind timer reset.",
    "notice.blindsLevelAdded": "Level {level} added.",
    "notice.blindsLevelSaved": "Level {level} saved.",
    "notice.blindsLevelDeleted": "Level {level} deleted.",
    "notice.blindsAllLevelsDeleted": "Blind structure deleted.",
    "hint.finishBlocked":
      "Cannot finish session yet: cash out or reverse operations until ON TABLE becomes 0.",
    "modal.startTitle": "Start Session",
    "modal.startDescription":
      "Start a new session with rate 1 {currencySymbol} = {chipRate} chips and BB {bigBlind}?",
    "modal.createPlayerTitle": "Create Player",
    "modal.createPlayerDescription": "Create player \"{name}\"?",
    "modal.confirmBuyInTitle": "Confirm Buy In",
    "modal.confirmBuyInDescription": "Add {chips} chips for {name}?",
    "modal.confirmCashOutTitle": "Confirm Cash Out",
    "modal.confirmCashOutDescription": "Cash out {chips} chips for {name}?",
    "modal.addPlayerTitle": "Add Player to Session",
    "modal.addPlayerDescription":
      "The player will appear in the session after the first buy in. This matches the current backend flow.",
    "modal.addToSession": "Buy-in",
    "modal.initialBuyIn": "Buy-in",
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
    "modal.deleteFinishTitle": "Cancel Finish",
    "modal.deleteFinishDescription":
      "Return this session to active status? This removes the finish state.",
    "modal.renamePlayerTitle": "Rename Player",
    "modal.editSessionConfigTitle": "Edit Session Settings",
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
    "error.failedDeleteFinish": "Failed to delete finish",
    "error.failedRenamePlayer": "Failed to rename player",
    "error.failedUpdateSessionConfig": "Failed to update session settings",
    "error.loginFailed": "Login failed",
    "error.registerFailed": "Registration failed",
    "error.logoutFailed": "Logout failed",
    "error.failedLoadAccount": "Failed to load account",
    "error.failedLoadAvailablePlayers": "Failed to load available players",
    "error.failedLinkPlayer": "Failed to link player",
    "error.failedUnlinkPlayer": "Failed to unlink player",
    "notice.playerDeleted": "Player deleted.",
    "notice.sessionDeleted": "Session deleted.",
    "notice.finishDeleted": "Finish deleted. Session is active again.",
    "notice.playerRenamed": "Player renamed.",
    "notice.sessionConfigUpdated": "Session settings updated.",
    "error.sessionNotBalanced":
      "Session is not balanced yet. Remaining chips on table: {chips}.",
    "error.invalid_request_id": "Request ID is missing.",
    "error.invalid_chips": "Enter a chip amount greater than zero.",
    "error.invalid_cash_out": "Cash out is not valid for the current table state.",
    "error.invalid_operation": "This operation cannot be performed.",
    "error.player_not_found": "Selected player does not exist.",
    "error.session_not_found": "Session was not found.",
    "error.invalid_credentials": "Invalid email or password.",
    "error.invalid_auth_email": "Enter a valid email.",
    "error.password_too_short": "Password must be at least 12 characters.",
    "error.user_already_exists": "This email is already registered.",
    "error.unauthorized": "Login required.",
    "error.forbidden": "Access denied.",
    "error.invalid_player_id": "Select a valid player.",
    "error.player_already_linked": "This player is already linked.",
    "error.user_player_not_linked": "This player is not linked to your account.",
    "error.rate_limited": "Too many login attempts. Try again later.",
    "error.session_not_active": "Session is not active anymore.",
    "error.session_finished": "Session is already finished.",
    "error.operation_not_found": "Operation was not found.",
    "error.operation_already_reversed": "Operation has already been reversed.",
    "error.internal_error": "Server returned an internal error.",
  },
  ru: {
    "app.title": "Poker Session Control",
    "app.subtitle": "Панель управления кэш-играми.",
    "nav.page": "Навигация по странице",
    "nav.backHome": "На главную",
    "nav.backSession": "К сессии",
    "auth.label": "Авторизация",
    "auth.email": "Email",
    "auth.password": "Пароль",
    "auth.login": "Войти",
    "auth.logout": "Выйти",
    "auth.signedIn": "Вход выполнен",
    "auth.noAccount": "Еще нет аккаунта?",
    "auth.register": "Зарегистрироваться",
    "guest.player": "Гость",
    "guest.noPlayer": "Игрок не выбран",
    "account.label": "Личный кабинет",
    "account.title": "Личный кабинет",
    "account.hint": "Можно привязать любого свободного игрока.",
    "account.loading": "Загружаем личный кабинет...",
    "account.loginRequired": "Войдите, чтобы управлять привязанными игроками.",
    "account.noLinkedPlayers": "Привязанных игроков пока нет.",
    "account.noAvailablePlayers": "Нет свободных игроков для привязки.",
    "account.selectPlayer": "Выберите игрока",
    "account.linkPlayer": "Привязать",
    "account.unlinkPlayer": "Отвязать",
    "lobby.connectTitle": "Открыть активную сессию",
    "lobby.latestActiveSession": "Последняя активная сессия",
    "lobby.openWorkspace": "Открыть сессию",
    "lobby.openBlindsClock": "Таймер блайндов",
    "lobby.startTitle": "Начать новую сессию",
    "lobby.chipRate": "Курс (1 {currencySymbol} за N фишек)",
    "lobby.currency": "Валюта",
    "lobby.bigBlind": "BB (размер большого блайнда)",
    "lobby.startSession": "Начать сессию",
    "lobby.addPlayerTitle": "Добавить нового игрока",
    "lobby.playerName": "Имя игрока",
    "lobby.playerNamePlaceholder": "Введите имя игрока",
    "lobby.createPlayer": "Создать игрока",
    "lobby.sessions": "Сессии",
    "lobby.players": "Игроки",
    "lobby.playersSort": "Сортировка",
    "language.title": "Язык интерфейса",
    "language.hint":
      "Русский выбирается автоматически для русскоязычных устройств. Для остальных используется английский.",
    "debug.deletePlayer": "Удалить игрока",
    "debug.deleteSession": "Удалить сессию",
    "debug.deleteFinish": "Отменить завершение",
    "debug.renamePlayer": "Переименовать игрока",
    "debug.editSessionConfig": "Изменить курс и большой блайнд",
    "session.title": "Сессия",
    "session.finish": "Завершить сессию",
    "session.chipRate": "Курс",
    "session.chipRateValue": "1 {currencySymbol} = {chips} фишек",
    "session.bigBlind": "BB",
    "session.bigBlindShort": "BB",
    "session.totalBuyIn": "Внесено",
    "session.totalCashOut": "Выведено",
    "session.onTable": "Фишек на столе",
    "session.totalMoneyIn": "Всего внесено деньгами",
    "session.addExistingPlayer": "Добавить игрока в сессию",
    "session.createNewPlayer": "Новый игрок",
    "session.players": "Игроки",
    "session.actions": "Действия",
    "session.actionsHint":
      "Cash-out доступен только игрокам, которые сейчас в игре.",
    "session.buyInHint":
      "Добавьте существующего игрока через Buy-in или создайте нового здесь.",
    "session.cashOutHint":
      "После кэшаута игрок считается завершившим игру. Для продолжения необходимо сделать ребай.",
    "session.player": "Игрок",
    "session.selectPlayer": "Выберите игрока",
    "session.chips": "Фишки",
    "session.buyIn": "Rebuy",
    "session.cashOut": "Cash-out",
    "session.latestOperations": "Последние операции",
    "blinds.title": "Таймер блайндов",
    "blinds.description":
      "Отдельный экран для турнирных блайндов с паузой, продолжением и редактированием будущих уровней.",
    "blinds.currentLevel": "Текущий уровень",
    "blinds.currentBlinds": "Текущие блайнды",
    "blinds.nextLevel": "Следующий уровень",
    "blinds.timeLeft": "Осталось времени",
    "blinds.controls": "Управление",
    "blinds.totalLevels": "Всего уровней",
    "blinds.upcomingLevels": "Впереди",
    "blinds.structure": "Структура блайндов",
    "blinds.selectedLevel": "Выбранный уровень",
    "blinds.addLevel": "Добавить уровень",
    "blinds.deleteAllLevels": "Удалить все уровни",
    "blinds.smallBlind": "SB",
    "blinds.bigBlind": "BB",
    "blinds.duration": "Минуты",
    "blinds.deleteLevel": "Удалить",
    "blinds.editHint":
      "Завершенные уровни заблокированы. Поставьте таймер на паузу, чтобы редактировать текущий или будущие уровни.",
    "blinds.maintenance": "Сервис",
    "blinds.resetHint": "Сброс возвращает таймер к первому уровню и останавливает отсчет.",
    "blinds.maintenanceHint":
      "Опасные действия вынесены отдельно от живого таймера, чтобы по ним было сложнее нажать случайно.",
    "blinds.toolStructure": "Структура",
    "blinds.toolClock": "Таймер",
    "blinds.toolDanger": "Опасная зона",
    "blinds.start": "Старт",
    "blinds.pause": "Пауза",
    "blinds.resume": "Продолжить",
    "blinds.reset": "Сброс",
    "blinds.noNextLevel": "Следующего уровня нет",
    "blinds.noLevels": "Уровни пока не добавлены",
    "blinds.levelCurrent": "текущий",
    "blinds.levelLocked": "заблокирован",
    "blinds.lockedWhileRunning": "Поставьте таймер на паузу, чтобы редактировать уровни.",
    "blinds.lockedCompletedLevel": "Завершенные уровни редактировать нельзя.",
    "blinds.statusStopped": "Остановлен",
    "blinds.statusRunning": "Идет",
    "blinds.statusPaused": "Пауза",
    "blinds.statusFinished": "Завершен",
    "blinds.levelValue": "Уровень {level}",
    "blinds.resetTitle": "Сбросить таймер",
    "blinds.resetDescription": "Сбросить таймер к первому уровню и остановить его?",
    "blinds.deleteLevelTitle": "Удалить уровень",
    "blinds.deleteLevelDescription": "Удалить уровень {level} из структуры?",
    "blinds.deleteAllLevelsTitle": "Удалить все уровни",
    "blinds.deleteAllLevelsDescription": "Удалить всю структуру блайндов?",
    "player.title": "Игрок",
    "player.dataPlaceholder": "Данные игрока появятся здесь",
    "player.from": "С",
    "player.to": "По",
    "player.applyPeriod": "Применить период",
    "player.allTime": "За все время",
    "player.period": "Период",
    "player.selectPeriod": "Выбрать период",
    "player.sessions": "Сессии",
    "player.totalBuyIn": "Всего внесено фишками",
    "player.totalCashOut": "Всего выведено фишками",
    "player.totalBuyInMoney": "Всего внесено деньгами",
    "player.totalCashOutMoney": "Всего выведено деньгами",
    "player.pnl": "PnL",
    "player.avgProfitPerSession": "Средний профит",
    "player.roi": "ROI",
    "player.avgBuyInPerSession": "Средний бай-ин",
    "player.profitMoney": "Профит деньгами",
    "player.hint.sessions": "Количество сессий в выбранном периоде.",
    "player.hint.totalBuyIn": "Сколько фишек игрок внес суммарно.",
    "player.hint.totalCashOut": "Сколько фишек игрок вывел суммарно.",
    "player.hint.totalBuyInMoney": "Сумма бай-инов в деньгах с пересчетом по курсу каждой сессии.",
    "player.hint.totalCashOutMoney": "Сумма кэш-аутов в деньгах с пересчетом по курсу каждой сессии.",
    "player.hint.pnl": "PnL = сумма кэш-аутов - сумма бай-инов.",
    "player.hint.avgProfitPerSession": "Средний профит на сессию = общий PnL / количество сессий.",
    "player.hint.roi": "ROI = общий PnL / сумма бай-инов * 100.",
    "player.hint.avgBuyInPerSession": "Средний бай-ин на сессию в фишках = сумма бай-инов фишками / количество сессий.",
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
    "common.close": "Закрыть",
    "common.save": "Сохранить",
    "common.noSessions": "Сессий нет",
    "common.noPlayers": "Игроков нет",
    "common.noOperations": "Операций пока нет",
    "common.noData": "Нет данных",
    "common.status": "Статус",
    "common.players": "Игроки",
    "common.sessions": "Сессии",
    "common.profit": "Профит",
    "common.buyIn": "Бай-ин",
    "common.totalBuyIn": "Бай-ин",
    "common.chips": "фишек",
    "common.cashOut": "Кэш-аут",
    "common.inGame": "В игре",
    "common.settled": "Рассчитан",
    "common.lastActivity": "Последняя активность",
    "sort.lastActivity": "По последней активности",
    "sort.sessionsCount": "По числу сессий",
    "sort.profit": "По профиту",
    "sort.name": "По имени",
    "status.active": "Активна",
    "status.finished": "Завершена",
    "operation.buy_in": "Бай-ин",
    "operation.cash_out": "Кэш-аут",
    "operation.reversal": "Отмена",
    "operation.finish": "Завершение сессии",
    "notice.noSession": "Нет сессии для открытия.",
    "notice.validChipRate": "Введите корректный курс.",
    "notice.validBigBlind": "Введите корректный большой блайнд.",
    "notice.sessionStarted": "Сессия начата.",
    "notice.authCredentialsRequired": "Введите email и пароль.",
    "notice.loginSuccess": "Вход выполнен.",
    "notice.logoutSuccess": "Вы вышли.",
    "notice.registrationPending": "Регистрация пока не подключена.",
    "notice.registrationSuccess": "Регистрация завершена.",
    "notice.selectAccountPlayer": "Выберите игрока для привязки.",
    "notice.accountPlayerLinked": "Игрок привязан к вашему аккаунту.",
    "notice.accountPlayerUnlinked": "Игрок отвязан от вашего аккаунта.",
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
    "notice.playerAlreadyCashedOut":
      "Этот игрок уже сделал кэш-аут. Если нужно, добавьте его заново через Buy-in.",
    "notice.enterPlayerAndBuyIn":
      "Введите имя игрока и корректный начальный бай-ин.",
    "notice.playerCreatedAndAdded": "Игрок {name} создан и добавлен в сессию.",
    "notice.cannotFinish":
      "Сессию пока нельзя завершить. Осталось фишек на столе: {chips}.",
    "notice.sessionFinished": "Сессия завершена.",
    "notice.operationReversed": "Операция отменена.",
    "notice.blindsStarted": "Таймер блайндов запущен.",
    "notice.blindsPaused": "Таймер блайндов поставлен на паузу.",
    "notice.blindsResumed": "Таймер блайндов продолжен.",
    "notice.blindsReset": "Таймер блайндов сброшен.",
    "notice.blindsLevelAdded": "Уровень {level} добавлен.",
    "notice.blindsLevelSaved": "Уровень {level} сохранен.",
    "notice.blindsLevelDeleted": "Уровень {level} удален.",
    "notice.blindsAllLevelsDeleted": "Структура блайндов удалена.",
    "hint.finishBlocked":
      "Сессию пока нельзя завершить: сделайте кэш-аут или отмените операции, пока НА СТОЛЕ не станет 0.",
    "modal.startTitle": "Начать сессию",
    "modal.startDescription":
      "Начать новую сессию с курсом 1 {currencySymbol} = {chipRate} фишек и BB {bigBlind}?",
    "modal.createPlayerTitle": "Создать игрока",
    "modal.createPlayerDescription": "Создать игрока «{name}»?",
    "modal.confirmBuyInTitle": "Подтвердить бай-ин",
    "modal.confirmBuyInDescription": "Добавить {chips} фишек для {name}?",
    "modal.confirmCashOutTitle": "Подтвердить кэш-аут",
    "modal.confirmCashOutDescription": "Вывести {chips} фишек для {name}?",
    "modal.addPlayerTitle": "Добавить игрока в сессию",
    "modal.addPlayerDescription":
      "Игрок появится в сессии после первого бай-ина. Это соответствует текущей логике бэкенда.",
    "modal.addToSession": "Buy-in",
    "modal.initialBuyIn": "Buy-in",
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
    "modal.deleteFinishTitle": "Отменить завершение",
    "modal.deleteFinishDescription":
      "Вернуть эту сессию в активный статус? Это удалит состояние завершения.",
    "modal.renamePlayerTitle": "Переименовать игрока",
    "modal.editSessionConfigTitle": "Изменить настройки сессии",
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
    "error.failedDeleteFinish": "Не удалось удалить завершение",
    "error.failedRenamePlayer": "Не удалось переименовать игрока",
    "error.failedUpdateSessionConfig": "Не удалось обновить настройки сессии",
    "error.loginFailed": "Не удалось войти",
    "error.registerFailed": "Не удалось зарегистрироваться",
    "error.logoutFailed": "Не удалось выйти",
    "error.failedLoadAccount": "Не удалось загрузить личный кабинет",
    "error.failedLoadAvailablePlayers": "Не удалось загрузить свободных игроков",
    "error.failedLinkPlayer": "Не удалось привязать игрока",
    "error.failedUnlinkPlayer": "Не удалось отвязать игрока",
    "notice.playerDeleted": "Игрок удален.",
    "notice.sessionDeleted": "Сессия удалена.",
    "notice.finishDeleted": "Завершение удалено. Сессия снова активна.",
    "notice.playerRenamed": "Игрок переименован.",
    "notice.sessionConfigUpdated": "Настройки сессии обновлены.",
    "error.sessionNotBalanced":
      "Сессия пока не сбалансирована. Осталось фишек на столе: {chips}.",
    "error.invalid_request_id": "Отсутствует ID запроса.",
    "error.invalid_chips": "Введите количество фишек больше нуля.",
    "error.invalid_cash_out": "Кэш-аут недопустим для текущего состояния стола.",
    "error.invalid_operation": "Эту операцию нельзя выполнить.",
    "error.player_not_found": "Выбранный игрок не найден.",
    "error.session_not_found": "Сессия не найдена.",
    "error.invalid_credentials": "Неверный email или пароль.",
    "error.invalid_auth_email": "Введите корректный email.",
    "error.password_too_short": "Пароль должен быть не короче 12 символов.",
    "error.user_already_exists": "Этот email уже зарегистрирован.",
    "error.unauthorized": "Нужно войти.",
    "error.forbidden": "Доступ запрещен.",
    "error.invalid_player_id": "Выберите корректного игрока.",
    "error.player_already_linked": "Этот игрок уже привязан.",
    "error.user_player_not_linked": "Этот игрок не привязан к вашему аккаунту.",
    "error.rate_limited": "Слишком много попыток входа. Попробуйте позже.",
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
