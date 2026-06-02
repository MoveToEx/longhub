import { configureStore } from "@reduxjs/toolkit";
import editShortcutDialogReducer from "@/features/images/state/edit-shortcut-dialog-slice";

export const store = configureStore({
  reducer: {
    editShortcutDialog: editShortcutDialogReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
export type AppStore = typeof store;