import React, { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import "./Homepage.css";

import { useJournal } from "./context/JournalContext";

const formatDate = (dateString) => {
  const [month, day, year] = dateString.split("/");
  return `${day}/${month}/${year}`;
};

const Home = (
  { entries, onAddEntry, onUpdateEntry, onDeleteEntry, onEntriesLoad },
) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [logoutConfirm, setLogoutConfirm] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState(null);
  const navigate = useNavigate();

  const { setSelectedEntry } = useJournal();

  // Load user info and entries on component mount
  useEffect(() => {
    const token = localStorage.getItem("token");
    const userData = localStorage.getItem("user");

    if (!token) {
      navigate("/login");
      return;
    }

    if (userData) {
      setUser(JSON.parse(userData));
    }

    loadEntries();
  }, [navigate]);

  const loadEntries = async () => {
    const token = localStorage.getItem("token");

    if (!token) {
      navigate("/login");
      return;
    }

    try {
      const response = await fetch("http://localhost:8080/api/entries", {
        method: "GET",
        headers: {
          "Authorization": `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });

      if (response.ok) {
        const entriesData = await response.json();
        if (onEntriesLoad) {
          onEntriesLoad(entriesData || []);
        }
      } else if (response.status === 401) {
        // Token expired
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
      } else {
        console.error("Failed to load entries");
      }
    } catch (error) {
      console.error("Error loading entries:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleEntryClick = (entry) => {
    setSelectedEntry(entry); // Store selected entry in JournalContext
    console.log("Entry Clicked in Homepage:", JSON.stringify(entry, null, 2));

    navigate(`/analysis/${entry.id}`);
  };

  const toggleSidebar = () => setSidebarOpen(!sidebarOpen);
  const toggleDarkMode = () => setIsDarkMode(!isDarkMode);

  //Sidebar auto-collapse
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (sidebarOpen) {
        const sidebar = document.querySelector(".sidebar");
        const toggleBtn = document.querySelector(".toggle-btn");

        // If clicked outside sidebar and toggle button, close the sidebar
        if (
          !sidebar.contains(event.target) && !toggleBtn.contains(event.target)
        ) {
          setSidebarOpen(false);
        }
      }
    };

    document.addEventListener("click", handleClickOutside);

    return () => {
      document.removeEventListener("click", handleClickOutside);
    };
  }, [sidebarOpen]);

  // Entry management functions for editing and deleting journal entries
  const handleEditEntry = async (index) => {
    const entry = filteredEntries[index]; // Use filteredEntries instead of entries

    // Prompt user to edit the selected entry
    const updatedTitle = prompt("Edit your title:", entry.title);
    if (updatedTitle === null) return; // User cancelled

    const updatedText = prompt("Edit your entry:", entry.text);
    if (updatedText === null) return; // User cancelled

    if (updatedTitle.trim() === "" || updatedText.trim() === "") {
      alert("Title and text cannot be empty");
      return;
    }

    const token = localStorage.getItem("token");
    if (!token) {
      navigate("/login");
      return;
    }

    try {
      const response = await fetch(
        `http://localhost:8080/api/entries/${entry.id}`,
        {
          method: "PUT",
          headers: {
            "Authorization": `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            title: updatedTitle.trim(),
            text: updatedText.trim(),
          }),
        },
      );

      if (response.ok) {
        const updatedEntry = await response.json();
        // Reload entries from server to ensure consistency
        await loadEntries();
      } else if (response.status === 401) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
      } else {
        alert("Failed to update entry");
      }
    } catch (error) {
      console.error("Error updating entry:", error);
      alert("Failed to update entry. Please try again.");
    }
  };

  const handleDeleteEntry = async (index) => {
    const entry = filteredEntries[index]; // Use filteredEntries instead of entries

    // Ask for confirmation before deleting an entry
    const confirmDelete = window.confirm(
      `Are you sure you want to delete "${entry.title}"?`,
    );

    if (!confirmDelete) return;

    const token = localStorage.getItem("token");
    if (!token) {
      navigate("/login");
      return;
    }

    try {
      const response = await fetch(
        `http://localhost:8080/api/entries/${entry.id}`,
        {
          method: "DELETE",
          headers: {
            "Authorization": `Bearer ${token}`,
          },
        },
      );

      if (response.ok) {
        // Reload entries from server to ensure consistency
        await loadEntries();
      } else if (response.status === 401) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        navigate("/login");
      } else {
        alert("Failed to delete entry");
      }
    } catch (error) {
      console.error("Error deleting entry:", error);
      alert("Failed to delete entry. Please try again.");
    }
  };

  const handleLogoutClick = () => setLogoutConfirm(true);
  const handleCancelLogout = () => setLogoutConfirm(false);

  const handleConfirmLogout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    alert("Logged Out Successfully!");
    setLogoutConfirm(false);
    navigate("/login");
  };

  // Ensure entries is always an array and remove duplicates based on ID
  const uniqueEntries = React.useMemo(() => {
    if (!Array.isArray(entries)) return [];

    // Remove duplicates based on entry ID
    const seen = new Set();
    return entries.filter((entry) => {
      if (seen.has(entry.id)) {
        return false;
      }
      seen.add(entry.id);
      return true;
    });
  }, [entries]);

  // Filters entries based on search query (matching date, title, or text)
  const filteredEntries = uniqueEntries.filter((entry) =>
    entry.text.toLowerCase().includes(searchQuery.toLowerCase()) ||
    entry.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    entry.date.includes(searchQuery)
  );

  if (loading) {
    return (
      <div className="loading-container">
        <h2>Loading your journal...</h2>
      </div>
    );
  }

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
            {user && <p className="user-greeting">Hello, {user.name}! 👋</p>}
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
              <li onClick={handleLogoutClick} style={{ cursor: "pointer" }}>
                🚪 Logout
              </li>
            </ul>
          </div>
        )}
      </aside>

      <main className="homepage-content">
        <h2>Welcome to My Journal</h2>
        {user && <p className="welcome-text">Good to see you, {user.name}!</p>}

        <div className="entry-box">
          <Link to="/AddEntry">
            <button>Add Entry</button>
          </Link>
        </div>

        <div className="entries">
          <h3>Your Journal Entries ({uniqueEntries.length})</h3>
          {filteredEntries.length > 0
            ? (
              <div className="entries-list">
                {filteredEntries.map((entry, index) => (
                  <div
                    key={entry.id || `entry-${index}`}
                    className="entry-item"
                    onClick={() => handleEntryClick(entry)}
                    style={{ cursor: "pointer" }}
                  >
                    <div className="entry-header">
                      <div className="entry-date-wrapper">
                        <span className="entry-date">
                          {formatDate(entry.date)}
                        </span>
                        <span className="entry-time">
                          {entry.time ? entry.time : "-"}
                        </span>
                      </div>

                      <h4 className="entry-title">{entry.title}</h4>

                      <div className="entry-text">
                        {entry.text}
                      </div>

                      <div
                        className="entry-actions"
                        style={{
                          display: "flex",
                          gap: "10px",
                          marginTop: "10px",
                        }}
                      >
                        <button
                          className="edit-btn"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleEditEntry(index);
                          }}
                          title="Edit entry"
                          aria-label={`Edit entry: ${entry.title}`}
                        >
                          ✏️
                        </button>
                        <button
                          className="delete-btn"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteEntry(index);
                          }}
                          title="Delete entry"
                          aria-label={`Delete entry: ${entry.title}`}
                        >
                          🗑️
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )
            : searchQuery
            ? (
              <div className="no-entries">
                <p>No entries found matching "{searchQuery}"</p>
              </div>
            )
            : (
              <div className="no-entries">
                <p>No entries yet. Create your first journal entry!</p>
              </div>
            )}
        </div>
      </main>

      {logoutConfirm && (
        <div className="logout-modal">
          <div className="logout-modal-content">
            <h3>Logout Confirmation</h3>
            <p>Are you sure you want to logout?</p>
            <div className="logout-modal-actions">
              <button className="yes-btn" onClick={handleConfirmLogout}>
                Yes, Logout
              </button>
              <button className="no-btn" onClick={handleCancelLogout}>
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const Homepage = (
  { entries, onAddEntry, onUpdateEntry, onDeleteEntry, onEntriesLoad },
) => {
  return (
    <Home
      entries={entries}
      onAddEntry={onAddEntry}
      onUpdateEntry={onUpdateEntry}
      onDeleteEntry={onDeleteEntry}
      onEntriesLoad={onEntriesLoad}
    />
  );
};

export default Homepage;
