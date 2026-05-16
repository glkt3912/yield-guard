"use client";

import React, { createContext, useContext } from "react";
import type { User } from "firebase/auth";
import { useAuth } from "@/hooks/useAuth";

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  linkWithEmail: (email: string, password: string) => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  linkWithEmail: async () => {},
});

export function useAuthContext(): AuthContextValue {
  return useContext(AuthContext);
}

interface RootLayoutClientProps {
  children: React.ReactNode;
}

export default function RootLayoutClient({ children }: RootLayoutClientProps) {
  const auth = useAuth();

  return <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>;
}
