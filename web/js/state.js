export const state = {
  authUiEnabled: false,
  debugMode: false,
  authUser: null,
  authChecked: false,
  authLoginOpen: false,
  accountPlayers: [],
  accountAvailablePlayers: [],
  accountLoading: false,
  guestPlayerId: "",
  guestPlayers: [],

  activeSessionId: "",
  session: null,

  overviewSessions: [],
  overviewPlayers: [],
  overviewPlayersAll: [],
  overviewPlayersSort: "last_activity",
  overviewPlayersShowAll: false,
  playersStatsRows: [],

  players: [],
  operations: [],
  expenses: [],
  settlementDrafts: {},
  settlementEditing: false,

  // player screen
  selectedPlayerId: "",
  selectedPlayerDetail: null,
  selectedPlayerFilters: {
    from: "",
    to: "",
  },
};
