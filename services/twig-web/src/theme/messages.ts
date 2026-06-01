export const messages = {
  emptyTaskList: "Nothing here yet — your future self is grateful. Add the first task.",
  emptyNameValidation: "A task needs a name to live by.",
  loginFailure: "That didn't match. Mind trying again?",
  sessionExpired: "Your session clocked out. Let's sign back in.",
  completeBlockedBySubtasks: "Hold on — finish its sub-tasks first.",
  reopenBlockedByParent: "Reopen its parent first to reopen this one.",
  taskNotFound: "That task seems to have wandered off.",
  connectivityError: "Couldn't reach the server. Want to try again?",
  saveSuccess: "Saved. ✨",
  allCompletedHidden: "All done! Your completed tasks are hiding — reveal them below.",
} as const;
