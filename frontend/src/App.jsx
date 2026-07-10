import { useState } from 'react';
import HomePage from './pages/HomePage';
import SwipePage from './pages/SwipePage';

function App() {
  const [currentPage, setCurrentPage] = useState('home'); // 'home' or 'swipe'

  if (currentPage === 'swipe') {
    return <SwipePage onBack={() => setCurrentPage('home')} />;
  }

  return <HomePage onStartSwipe={() => setCurrentPage('swipe')} />;
}

export default App;
