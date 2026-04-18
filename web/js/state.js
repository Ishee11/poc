export const state = {
  debugMode: false,

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
