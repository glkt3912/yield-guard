"use client";

import { useEffect, useState } from "react";
import {
  signInAnonymously,
  linkWithCredential,
  EmailAuthProvider,
  onAuthStateChanged,
  type User,
} from "firebase/auth";
import { getFirebaseAuth } from "@/lib/firebase";

interface UseAuthReturn {
  loading: boolean;
  user: User | null;
  linkWithEmail: (email: string, password: string) => Promise<void>;
}

export function useAuth(): UseAuthReturn {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(() => getFirebaseAuth() !== null);

  useEffect(() => {
    const auth = getFirebaseAuth();

    if (!auth) return;

    const unsubscribe = onAuthStateChanged(auth, (currentUser) => {
      setUser(currentUser);
      setLoading(false);

      if (!currentUser) {
        signInAnonymously(auth).catch((err) => {
          console.error("[useAuth] anonymous sign-in failed:", err);
        });
      }
    });

    return () => unsubscribe();
  }, []);

  async function linkWithEmail(email: string, password: string): Promise<void> {
    const auth = getFirebaseAuth();
    if (!auth) throw new Error("Firebase が設定されていません");
    if (!user) throw new Error("ユーザーが初期化されていません");
    const credential = EmailAuthProvider.credential(email, password);
    await linkWithCredential(user, credential);
  }

  return { loading, user, linkWithEmail };
}
