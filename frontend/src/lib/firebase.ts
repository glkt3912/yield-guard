import { getApps, initializeApp, type FirebaseApp } from "firebase/app";
import { getAuth, type Auth } from "firebase/auth";
import { getFirestore, type Firestore } from "firebase/firestore";

const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
};

// Firebase is only initializable on the client side with valid config.
// When env vars are absent (build time / SSR prerender) we skip initialization.
const isConfigured = Boolean(
  firebaseConfig.apiKey &&
  firebaseConfig.authDomain &&
  firebaseConfig.projectId &&
  firebaseConfig.appId
);

function getApp(): FirebaseApp | null {
  if (!isConfigured) return null;
  return getApps().length > 0 ? getApps()[0] : initializeApp(firebaseConfig);
}

// Lazy singletons — safe to call from "use client" components
export function getFirebaseAuth(): Auth | null {
  const app = getApp();
  if (!app) return null;
  return getAuth(app);
}

export function getFirebaseDb(): Firestore | null {
  const app = getApp();
  if (!app) return null;
  return getFirestore(app);
}

export { isConfigured };
