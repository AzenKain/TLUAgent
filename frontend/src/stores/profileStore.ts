import { create } from 'zustand';

// ProfileMessage represents user feedback alerts for profile actions.
export interface ProfileMessage {
  text: string;
  type: string;
}

// ProfileState defines the state and actions for the profile and password management screen.
export interface ProfileState {
  fullName: string;
  studentCode: string;
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
  profileMsg: ProfileMessage;
  passwordMsg: ProfileMessage;

  setFullName: (name: string) => void;
  setStudentCode: (code: string) => void;
  setOldPassword: (password: string) => void;
  setNewPassword: (password: string) => void;
  setConfirmPassword: (password: string) => void;
  setProfileMsg: (msg: ProfileMessage) => void;
  setPasswordMsg: (msg: ProfileMessage) => void;
  resetPasswordForm: () => void;
}

// useProfileStore manages state for user profile editing and password change.
export const useProfileStore = create<ProfileState>((set) => ({
  fullName: '',
  studentCode: '',
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
  profileMsg: { text: '', type: '' },
  passwordMsg: { text: '', type: '' },

  setFullName: (fullName) => set({ fullName }),
  setStudentCode: (studentCode) => set({ studentCode }),
  setOldPassword: (oldPassword) => set({ oldPassword }),
  setNewPassword: (newPassword) => set({ newPassword }),
  setConfirmPassword: (confirmPassword) => set({ confirmPassword }),
  setProfileMsg: (profileMsg) => set({ profileMsg }),
  setPasswordMsg: (passwordMsg) => set({ passwordMsg }),
  resetPasswordForm: () =>
    set({
      oldPassword: '',
      newPassword: '',
      confirmPassword: '',
    }),
}));
