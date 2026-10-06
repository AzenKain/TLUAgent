import { create } from 'zustand';

// AuthFormState manages form inputs and validation errors for authentication and setup pages.
export interface AuthFormState {
  loginEmail: string;
  loginPassword: string;
  loginError: string;

  setupEmail: string;
  setupPassword: string;
  setupFullName: string;
  setupStudentCode: string;
  setupError: string;

  setLoginEmail: (email: string) => void;
  setLoginPassword: (password: string) => void;
  setLoginError: (error: string) => void;
  resetLoginForm: () => void;

  setSetupEmail: (email: string) => void;
  setSetupPassword: (password: string) => void;
  setSetupFullName: (fullName: string) => void;
  setSetupStudentCode: (studentCode: string) => void;
  setSetupError: (error: string) => void;
  resetSetupForm: () => void;
}

// useAuthFormStore provides Zustand state management for login and initial setup forms.
export const useAuthFormStore = create<AuthFormState>((set) => ({
  loginEmail: '',
  loginPassword: '',
  loginError: '',

  setupEmail: '',
  setupPassword: '',
  setupFullName: '',
  setupStudentCode: '',
  setupError: '',

  setLoginEmail: (loginEmail) => set({ loginEmail }),
  setLoginPassword: (loginPassword) => set({ loginPassword }),
  setLoginError: (loginError) => set({ loginError }),
  resetLoginForm: () => set({ loginEmail: '', loginPassword: '', loginError: '' }),

  setSetupEmail: (setupEmail) => set({ setupEmail }),
  setSetupPassword: (setupPassword) => set({ setupPassword }),
  setSetupFullName: (setupFullName) => set({ setupFullName }),
  setSetupStudentCode: (setupStudentCode) => set({ setupStudentCode }),
  setSetupError: (setupError) => set({ setupError }),
  resetSetupForm: () =>
    set({
      setupEmail: '',
      setupPassword: '',
      setupFullName: '',
      setupStudentCode: '',
      setupError: '',
    }),
}));
