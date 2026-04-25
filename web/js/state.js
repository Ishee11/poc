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
  overviewPlayersSort: "last_activity",

  players: [],
  operations: [],

  // player screen
  selectedPlayerId: "",
  selectedPlayerDetail: null,
  selectedPlayerFilters: {
    from: "",
    to: "",
  },
};
