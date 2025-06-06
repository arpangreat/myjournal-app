import React, { useEffect, useState } from "react";
import { Route, Routes, useNavigate } from "react-router-dom";
import Homepage from "./Homepage";
import "./App.css";
import AddEntry from "./AddEntry";
import { authAPI, entriesAPI, handleAPIError } from "./api";

import { JournalProvider } from "./context/JournalContext";
import Analysis from "./Analysis"; 

const AuthPage = () => {
  const [isSignup, setIsSignup] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (event) => {
    event.preventDefault();
    setLoading(true);
    try {
      if (isSignup) {
        console.log("Signing up with:", name, email, password);
        await authAPI.signup(name, email, password);
      } else {
        await authAPI.login(email, password);
      }
      navigate("/Homepage");
    } catch (error) {
      alert(handleAPIError(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container">
      <h2>My Journals!!!</h2>
      <form onSubmit={handleSubmit}>
        {isSignup && (
          <div>
            <label>Name:</label>
            <input
              type="text"
              placeholder="Enter your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              disabled={loading}
            />
          </div>
        )}
        <div>
          <label>Email:</label>
          <input
            type="email"
            placeholder="example@domain.com...."
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            disabled={loading}
          />
        </div>
        <div className="input-container">
          <label className="password-label">Password:</label>
          <div className="password-container">
            <input
              type={showPassword ? "text" : "password"}
              placeholder="Enter your password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={loading}
              className="password-input"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              disabled={loading}
              className="toggle-password"
            >
              {showPassword ? "Hide" : "Show"}
            </button> 
          </div>   
        </div>
        <button type="submit" disabled={loading}>
          {loading ? "Loading..." : (isSignup ? "Sign Up" : "Login")}
        </button>
      </form>
      <p>
        {isSignup ? "Already have an account?" : "Don't have an account?"}{" "}
        <button
          onClick={() => setIsSignup(!isSignup)}
          disabled={loading}
          type="button"
        >
          {isSignup ? "Login" : "Sign Up"}
        </button>
      </p>
    </div>
  );
};

const AppContent = () => {
  const [entries, setEntries] = useState([]);

  const handleEntriesLoad = (loadedEntries) => {
    setEntries(loadedEntries);
  };

  const handleAddEntry = async (newEntry) => {
    try {
      // If API is available, create entry via API
      if (typeof entriesAPI !== "undefined") {
        const createdEntry = await entriesAPI.createEntry(
          newEntry.title,
          newEntry.text,
          newEntry.date,
        );
        console.log("Entry created:", createdEntry);
        // Don't manually update state here - let the Homepage component reload entries
        // This prevents duplicates
        return createdEntry;
      } else {
        // Fallback to local state if API not available
        console.log("Adding new entry:", newEntry);
        setEntries((prevEntries) => [newEntry, ...prevEntries]);
        return newEntry;
      }
    } catch (error) {
      console.error("Error adding entry:", error);
      alert(handleAPIError(error));
      throw error;
    }
  };

  const handleUpdateEntry = async (index, updatedEntry) => {
    try {
      if (typeof entriesAPI !== "undefined") {
        const entryToUpdate = entries[index];
        const updatedFromAPI = await entriesAPI.updateEntry(
          entryToUpdate.id,
          updatedEntry.title,
          updatedEntry.text,
        );
        // Don't manually update state - let Homepage component reload entries
        return updatedFromAPI;
      } else {
        // Fallback to local state
        const updatedEntries = [...entries];
        updatedEntries[index] = updatedEntry;
        setEntries(updatedEntries);
      }
    } catch (error) {
      alert(handleAPIError(error));
    }
  };

  const handleDeleteEntry = async (index) => {
    try {
      if (typeof entriesAPI !== "undefined") {
        const entryToDelete = entries[index];
        await entriesAPI.deleteEntry(entryToDelete.id);
        // Don't manually update state - let Homepage component reload entries
      } else {
        // Fallback to local state
        setEntries(entries.filter((_, i) => i !== index));
      }
    } catch (error) {
      alert(handleAPIError(error));
    }
  };

  return (
    <Routes>
      <Route path="/" element={<AuthPage />} />
      <Route path="/login" element={<AuthPage />} />
      <Route
        path="/Homepage"
        element={
          <Homepage
            entries={entries}
            onAddEntry={handleAddEntry}
            onUpdateEntry={handleUpdateEntry}
            onDeleteEntry={handleDeleteEntry}
            onEntriesLoad={handleEntriesLoad}
          />
        }
      />
      <Route
        path="/AddEntry"
        element={<AddEntry onAddEntry={handleAddEntry} />}
      />
      <Route path="/analysis" element={<Analysis />} />
    </Routes>
  );
};

const App = () => {
  return (
    <JournalProvider>
      <AppContent />
    </JournalProvider>
  );
};

export default App;
