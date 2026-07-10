// Keeps track of movies that already won a party, so they don't get
// proposed again. Saved in localStorage so it survives page reloads.
const STORAGE_KEY = 'watchparty_already_picked';

export function getExcludedMovieIds() {
  const saved = localStorage.getItem(STORAGE_KEY);
  return saved ? JSON.parse(saved) : [];
}

export function addExcludedMovieId(movieId) {
  const excludedIds = getExcludedMovieIds();
  if (!excludedIds.includes(movieId)) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify([...excludedIds, movieId]));
  }
}
