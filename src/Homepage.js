import React, { useState } from "react";
import "./Homepage.css";

const Homepage = () => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const toggleSidebar = () => {
    setSidebarOpen(!sidebarOpen);
  };

  const [entries, setEntries] = useState([]);
  const [newEntry, setNewEntry] = useState("");

  const handleAddEntry = () => {
    if (newEntry.trim() !== "") {
      setEntries([...entries, newEntry]);
      setNewEntry("");
    }
  };

  return (
    <div className="main-container">
      {/* Expandable Sidebar Dashboard */}
      <aside className={`sidebar ${sidebarOpen ? "expanded" : ""}`}>
        <button className="toggle-btn" onClick={toggleSidebar}>
          {sidebarOpen ? "✖ Close" : "☰ Menu"}
        </button>
        {sidebarOpen && (
          <div className="sidebar-content">
            <h2>Dashboard</h2>
            <ul>
              <li>🏠 Home</li>
              <li>📝 New Entry</li>
              <li>📜 View Entries</li>
              <li>⚙️ Settings</li>
            </ul>
          </div>
        )}
      </aside>

      {/* Journal Area */}
      <main className={`homepage-content ${sidebarOpen ? "shifted" : ""}`}>
        <h2>Welcome to My Journal</h2>
        <div className="entry-box">
          <textarea
            placeholder="Write your thoughts ..."
            value={newEntry}
            onChange={(e) => setNewEntry(e.target.value)}
          />
          <button onClick={handleAddEntry}>Add Entry</button>
        </div>
        <div className="entries">
          <h3>Your Journal Entries</h3>
          {entries.length > 0 ? (
            <ul>
              {entries.map((entry, index) => (
                <li key={index}>{entry}</li>
              ))}
            </ul>
          ) : (
            <p>No entries yet ... Start writing!</p>
          )}
        </div>
      </main>
    </div>
  );
};

export default Homepage;
