import React, { useState } from "react";
import { Link } from "react-router-dom";
import { Route, Routes, useNavigate } from "react-router-dom";
import AddEntry from "./AddEntry";
import "./Homepage.css";

const Home = ({ entries, onAddEntry, onUpdateEntry, onDeleteEntry }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [newEntry, setNewEntry] = useState("");
  const [logoutConfirm, setLogoutConfirm] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const navigate = useNavigate();

  const toggleSidebar = () => setSidebarOpen(!sidebarOpen);
  const toggleDarkMode = () => setIsDarkMode(!isDarkMode);

  // Entry management functions for editing and deleting journal entries
  const handleEditEntry = (index) => {
    // Prompt user to edit the selected entry
    const updatedText = prompt("Edit your entry:", entries[index].text);
    const updatedTitle = prompt("Edit your title:", entries[index].title);
    // If the user provides a valid input, update the entry
    if (updatedText !== null && updatedTitle !== null) {
      const updatedEntry = {
        ...entries[index],
        text: updatedText,
        title: updatedTitle,
      };
      onUpdateEntry(index, updatedEntry);
    }
  };

  const handleDeleteEntry = (index) => {
    // Ask for confirmation before deleting an entry
    const confirmDelete = window.confirm("Are you sure?");
    // If confirmed, remove the entry from the list
    if (confirmDelete) {
      onDeleteEntry(index);
    }
  };

  const handleLogoutClick = () => setLogoutConfirm(true);
  const handleCancelLogout = () => setLogoutConfirm(false);
  const handleConfirmLogout = () => {
    alert("Logged Out Successfully!");
    setLogoutConfirm(false);
  };

  // Filters entries based on search query (matching date or text)
  const filteredEntries = entries.filter((entry) =>
    entry.text.toLowerCase().includes(searchQuery.toLowerCase()) ||
    entry.date.includes(searchQuery)
  );

  return (
    <div className={`main-container ${isDarkMode ? "dark-mode" : ""}`}>
      <div className={`search-container ${searchOpen ? "open" : ""}`}>
        <div className="search-input-wrapper">
          <input
            type="text"
            className="search-input"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search journal..."
          />
          {/* Clear button - only show when there's text and search is open */}
          {searchQuery && searchOpen && (
            <button
              onClick={() => setSearchQuery("")}
              className="search-clear-btn"
              title="Clear search"
              aria-label="Clear search"
            >
              ×
            </button>
          )}
        </div>
        <button
          className="search-icon"
          onClick={() => setSearchOpen(!searchOpen)}
          title={searchOpen ? "Close search" : "Open search"}
        >
          🔍
        </button>
      </div>

      <aside className={`sidebar ${sidebarOpen ? "expanded" : ""}`}>
        <button className="toggle-btn" onClick={toggleSidebar}>
          {sidebarOpen ? "✖ Close" : "☰"}
        </button>
        {sidebarOpen && (
          <div className="sidebar-content">
            <h2>Dashboard</h2>
            <ul>
              <li>🔒 Privacy Settings</li>
              <li className="dark-mode-toggle">
                🌙 Dark Mode
                <label className="switch">
                  <input
                    type="checkbox"
                    checked={isDarkMode}
                    onChange={toggleDarkMode}
                  />
                  <span className="slider"></span>
                </label>
              </li>
              <li>📜 Terms & Policy</li>
              <li onClick={handleLogoutClick}>🚪 Logout</li>
            </ul>
          </div>
        )}
      </aside>

      <main className="homepage-content">
        <h2>Welcome to My Journal</h2>
        <div className="entry-box">
          <Link to="/AddEntry">
            <button>Add Entry</button>
          </Link>
        </div>
        <div className="entries">
          <h3>Your Journal Entries</h3>
          {filteredEntries.length > 0
            ? (
              <ul>
                {filteredEntries.map((entry, index) => (
                  <li key={index}>
                    <strong>{entry.date}:</strong>
                    <br />
                    <strong>{entry.title}</strong>
                    <br />
                    {entry.text.replace(
                      new RegExp(`(${searchQuery})`, "gi"),
                      "$1",
                    )}
                    <div className="entry-actions">
                      <button
                        className="edit-btn"
                        onClick={() => handleEditEntry(index)}
                      >
                        ✏️
                      </button>
                      <button
                        className="delete-btn"
                        onClick={() => handleDeleteEntry(index)}
                      >
                        🗑️
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )
            : <p>No entries....</p>}
        </div>
      </main>

      {logoutConfirm && (
        <div className="logout-modal">
          <h3>Logout!</h3>
          <h3>Are you sure?</h3>
          <button className="yes-btn" onClick={handleConfirmLogout}>
            Yes
          </button>
          <button className="no-btn" onClick={handleCancelLogout}>
            No
          </button>
        </div>
      )}
    </div>
  );
};

const Homepage = ({ entries, onAddEntry, onUpdateEntry, onDeleteEntry }) => {
  // Remove the local state and nested Routes since we're handling this in App.js now
  return (
    <Home
      entries={entries}
      onAddEntry={onAddEntry}
      onUpdateEntry={onUpdateEntry}
      onDeleteEntry={onDeleteEntry}
    />
  );
};

export default Homepage;
