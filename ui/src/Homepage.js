import React, { useState } from "react";
import { Link, Route, Routes,useNavigate } from "react-router-dom";
import AddEntry from "./AddEntry";
import "./Homepage.css";

const Home = () => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [entries, setEntries] = useState([]); // Empty array, users will add entries
  const [newEntry, setNewEntry] = useState("");
  const [logoutConfirm, setLogoutConfirm] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const navigate = useNavigate("/AddEntry");

  const toggleSidebar = () => setSidebarOpen(!sidebarOpen);
  const toggleDarkMode = () => setIsDarkMode(!isDarkMode);

  /*const handleAddEntry = () => {
    if (newEntry.trim() !== "") {
      const currentDate = new Date().toISOString().split("T")[0]; // Get today's date
      setEntries([{ date: currentDate, text: newEntry }, ...entries]); // Prepend new entry to the list
      setNewEntry(""); // Clear input after saving
    }
  };*/

  // Entry management functions for editing and deleting journal entries

  const handleEditEntry = (index) => {
    // Prompt user to edit the selected entry
    const updatedText = prompt("Edit your entry:", entries[index].text);

    // If the user provides a valid input, update the entry
    if (updatedText !== null) {
      const updatedEntries = [...entries]; // Copy existing entries
      updatedEntries[index].text = updatedText; // Modify the selected entry
      setEntries(updatedEntries); // Update the state with edited entries
    }
  };

  const handleDeleteEntry = (index) => {
    // Ask for confirmation before deleting an entry
    const confirmDelete = window.confirm("Are you sure?");

    // If confirmed, remove the entry from the list
    if (confirmDelete) {
      setEntries(entries.filter((_, i) => i !== index)); // Keep only the entries that don't match the index
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
          <button>Add Entry </button>
        </Link>
          
        </div>
        <div className="entries">
          <h3>Your Journal Entries</h3>
          <main>
            {filteredEntries.length > 0 ? (
              <ul>
                {filteredEntries.map((entry, index) => (
                  <li key={index}>
                    <strong>{entry.date}:</strong><br /> {entry.text.replace(new RegExp(`(${searchQuery})`, "gi"), "$1")}
                    <div className="entry-actions">
                      <button className="edit-btn" onClick={() => handleEditEntry(index)}>✏️</button>
                      <button className="delete-btn" onClick={() => handleDeleteEntry(index)}>🗑️</button>
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <p>No entries.... </p>
            )}
          </main>
        </div>
      </main>

      {logoutConfirm && (
        <div className="logout-modal">
          <h3>Logout!</h3>
          <h3>Are you sure?</h3>
          <button className="yes-btn" onClick={handleConfirmLogout}>Yes</button>
          <button className="no-btn" onClick={handleCancelLogout}>No</button>
        </div>
      )}
    </div>
  );
};

const Homepage = () => {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/AddEntry" element={<AddEntry />} />
    </Routes>
  );
};

export default Homepage;
