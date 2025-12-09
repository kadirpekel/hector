export function getBaseUrl(): string {
    // Break circular dependency by not importing store here.
    // Ideally this should be passed in, but for now defaulting to window location
    // is safe as the store initializes this way too.
    return typeof window !== "undefined" ? window.location.origin : "";
}

