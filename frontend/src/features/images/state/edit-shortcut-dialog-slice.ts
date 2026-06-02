import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export type EditShortcutDialogPayload = {
  id: number,
  shortcut: string,
}

type EditShortcutDialogState = {
  open: boolean,
  payload: EditShortcutDialogPayload | null,
}

const initialState: EditShortcutDialogState = {
  open: false,
  payload: null,
}

const editShortcutDialogSlice = createSlice({
  name: "editShortcutDialog",
  initialState,
  reducers: {
    openEditShortcutDialog(state, action: PayloadAction<EditShortcutDialogPayload>) {
      state.open = true;
      state.payload = action.payload;
    },
    closeEditShortcutDialog(state) {
      state.open = false;
      state.payload = null;
    },
  },
});

export const {
  openEditShortcutDialog,
  closeEditShortcutDialog,
} = editShortcutDialogSlice.actions;

export default editShortcutDialogSlice.reducer;
