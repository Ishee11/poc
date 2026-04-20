export const state = {
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
