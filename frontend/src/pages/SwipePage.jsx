import { useRef, useState } from 'react';
import './SwipePage.css';
import { movies as allMovies } from '../data/movies';
import { getExcludedMovieIds, addExcludedMovieId } from '../utils/watchedMovies';

const DECK_SIZE = 10; // how many movies the group swipes on per party
const FRIEND_COUNT = 3; // simulated friends swiping alongside you
const SWIPE_THRESHOLD = 100; // how far (in px) you need to drag to count as a swipe

// Generates a simple placeholder poster with the movie title on it.
function getPosterUrl(title) {
  return `https://placehold.co/400x600/1c1c1f/f5f5f7?text=${encodeURIComponent(title)}`;
}

// Picks up to DECK_SIZE movies that haven't already won a past party,
// and gives each one a random number of "friend" likes to simulate
// the rest of the group swiping at the same time.
function buildDeck() {
  const excludedIds = getExcludedMovieIds();
  const available = allMovies.filter((movie) => !excludedIds.includes(movie.id));
  const shuffled = [...available].sort(() => Math.random() - 0.5);

  return shuffled.slice(0, DECK_SIZE).map((movie) => ({
    ...movie,
    friendLikes: Math.floor(Math.random() * (FRIEND_COUNT + 1)),
  }));
}

// The winner is whichever movie in the deck has the most total likes,
// counting the simulated friends plus you.
function pickWinner(deck, likedMovieIds) {
  let winner = null;

  for (const movie of deck) {
    const totalLikes = movie.friendLikes + (likedMovieIds.includes(movie.id) ? 1 : 0);
    if (!winner || totalLikes > winner.totalLikes) {
      winner = { ...movie, totalLikes };
    }
  }

  return winner;
}

function SwipePage({ onBack }) {
  const [deck, setDeck] = useState(buildDeck);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [likedMovieIds, setLikedMovieIds] = useState([]);
  const [dragX, setDragX] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const dragStartXRef = useRef(0);

  const isFinished = currentIndex >= deck.length;
  const currentMovie = deck[currentIndex];
  const winner = isFinished && deck.length > 0 ? pickWinner(deck, likedMovieIds) : null;

  function handleSwipe(liked) {
    if (liked) {
      setLikedMovieIds((previousIds) => [...previousIds, currentMovie.id]);
    }
    setCurrentIndex((previousIndex) => previousIndex + 1);
    setDragX(0);
  }

  function handlePointerDown(event) {
    event.currentTarget.setPointerCapture(event.pointerId);
    setIsDragging(true);
    dragStartXRef.current = event.clientX;
  }

  function handlePointerMove(event) {
    if (!isDragging) return;
    setDragX(event.clientX - dragStartXRef.current);
  }

  function handlePointerUp() {
    if (!isDragging) return;
    setIsDragging(false);

    if (dragX > SWIPE_THRESHOLD) {
      handleSwipe(true);
    } else if (dragX < -SWIPE_THRESHOLD) {
      handleSwipe(false);
    } else {
      setDragX(0);
    }
  }

  function handleNewParty() {
    if (winner) {
      addExcludedMovieId(winner.id);
    }
    setDeck(buildDeck());
    setCurrentIndex(0);
    setLikedMovieIds([]);
  }

  return (
    <div className="swipe-page">
      <div className="swipe-header">
        <button className="back-button" onClick={onBack}>
          ← Back
        </button>
        <h1>Pick a Movie</h1>
      </div>

      {deck.length === 0 ? (
        <div className="result-card">
          <p>You've already picked every movie we know! 🍿</p>
        </div>
      ) : !isFinished ? (
        <>
          <div
            className="movie-card"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            style={{
              transform: `translateX(${dragX}px) rotate(${dragX / 20}deg)`,
              transition: isDragging ? 'none' : 'transform 0.3s ease',
            }}
          >
            <span
              className="like-stamp"
              style={{ opacity: Math.min(Math.max(dragX / SWIPE_THRESHOLD, 0), 1) }}
            >
              LIKE
            </span>
            <span
              className="nope-stamp"
              style={{ opacity: Math.min(Math.max(-dragX / SWIPE_THRESHOLD, 0), 1) }}
            >
              NOPE
            </span>
            <img
              className="movie-poster"
              src={getPosterUrl(currentMovie.title)}
              alt={currentMovie.title}
              draggable="false"
            />
            <div className="movie-info">
              <h2>{currentMovie.title}</h2>
              <p>{currentMovie.genre} · {currentMovie.year}</p>
            </div>
          </div>

          <div className="swipe-buttons">
            <button
              className="dislike-button"
              onClick={() => handleSwipe(false)}
              aria-label="Dislike this movie"
            >
              ✕
            </button>
            <button
              className="like-button"
              onClick={() => handleSwipe(true)}
              aria-label="Like this movie"
            >
              ♥
            </button>
          </div>

          <p className="swipe-progress">
            {currentIndex + 1} / {deck.length}
          </p>
        </>
      ) : (
        <div className="result-card">
          <p className="result-label">Tonight's pick</p>
          <img
            className="result-poster"
            src={getPosterUrl(winner.title)}
            alt={winner.title}
          />
          <h2>{winner.title}</h2>
          <p className="result-votes">
            {winner.totalLikes} of {FRIEND_COUNT + 1} people want to watch this
          </p>
          <button className="restart-button" onClick={handleNewParty}>
            Start a New Party
          </button>
        </div>
      )}
    </div>
  );
}

export default SwipePage;
