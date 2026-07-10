import { useState } from 'react';
import './HomePage.css';

// Fake data for now. Later this will come from the backend API.
const initialParties = [
  { id: 1, name: 'Friday Movie Night', date: 'Fri, July 12 - 8:00 PM', members: 4 },
  { id: 2, name: 'Horror Marathon', date: 'Sat, July 13 - 9:30 PM', members: 6 },
  { id: 3, name: 'Sunday Chill Session', date: 'Sun, July 14 - 6:00 PM', members: 3 },
];

function HomePage({ onStartSwipe }) {
  const [parties, setParties] = useState(initialParties);

  function handleCreateParty() {
    const name = window.prompt('Name your watch party:');
    if (!name) return;

    const newParty = {
      id: Date.now(),
      name,
      date: 'Date to be decided',
      members: 1,
    };
    setParties((previousParties) => [newParty, ...previousParties]);
  }

  return (
    <div className="home-page">
      <div className="home-header">
        <h1>Your Watch Parties</h1>
        <button className="new-party-button" onClick={handleCreateParty}>
          + New Party
        </button>
      </div>

      <div className="party-list">
        {parties.map((party) => (
          <div className="party-card" key={party.id}>
            <div className="party-info">
              <h2>{party.name}</h2>
              <p className="party-date">{party.date}</p>
              <p className="party-members">{party.members} friends joined</p>
            </div>
            <button className="enter-button" onClick={onStartSwipe}>
              Pick a movie
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

export default HomePage;
