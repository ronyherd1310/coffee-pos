export type AuthSessionState =
  | { status: "loading" }
  | { status: "authenticated" }
  | { status: "unauthenticated" }
  | { status: "unavailable" };
