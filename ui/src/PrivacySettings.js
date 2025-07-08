// PrivacySettings.js
import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import "./PrivacySettings.css"; // Optional: for styles

const PrivacySettings = () => {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  // Password visibility states
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  // Dark mode state
  const [isDarkMode, setIsDarkMode] = useState(() => {
    const savedDarkMode = localStorage.getItem("darkMode");
    return savedDarkMode === "true";
  });

  // Apply dark mode styles to document body
  useEffect(() => {
    if (isDarkMode) {
      document.body.classList.add('dark-mode');
    } else {
      document.body.classList.remove('dark-mode');
    }
  }, [isDarkMode]);

  // Listen for dark mode changes from localStorage
  useEffect(() => {
    const handleStorageChange = (e) => {
      if (e.key === "darkMode") {
        setIsDarkMode(e.newValue === "true");
      }
    };

    window.addEventListener("storage", handleStorageChange);

    // Also check for changes periodically (in case user changes dark mode in same tab)
    const interval = setInterval(() => {
      const currentDarkMode = localStorage.getItem("darkMode") === "true";
      if (currentDarkMode !== isDarkMode) {
        setIsDarkMode(currentDarkMode);
      }
    }, 1000);

    return () => {
      window.removeEventListener("storage", handleStorageChange);
      clearInterval(interval);
    };
  }, [isDarkMode]);



  // Fetch current user data on component mount
  useEffect(() => {
    fetchUserData();
  }, []);

  const fetchUserData = async () => {
    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }

      const response = await fetch("http://localhost:8080/api/user/profile", {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (response.ok) {
        const userData = await response.json();
        setName(userData.name);
        setEmail(userData.email);
      } else if (response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login");
      } else {
        setError("Failed to fetch user data");
      }
    } catch (err) {
      setError("Failed to connect to server");
    }
  };

  const validateForm = () => {
    if (!name.trim()) {
      setError("Name is required");
      return false;
    }

    if (newPassword || confirmPassword) {
      if (!currentPassword) {
        setError("Current password is required to change password");
        return false;
      }
      if (newPassword !== confirmPassword) {
        setError("New passwords do not match");
        return false;
      }
      if (newPassword.length < 6) {
        setError("New password must be at least 6 characters long");
        return false;
      }
    }

    return true;
  };

  const handleSave = async () => {
    setError("");
    setMessage("");

    if (!validateForm()) {
      return;
    }

    setLoading(true);

    try {
      const token = localStorage.getItem("token");
      if (!token) {
        navigate("/login");
        return;
      }

      const updateData = {
        name: name.trim(),
      };

      // Only include password fields if user wants to change password
      if (newPassword) {
        updateData.currentPassword = currentPassword;
        updateData.newPassword = newPassword;
      }

      const response = await fetch("http://localhost:8080/api/user/profile", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(updateData),
      });

      if (response.ok) {
        setMessage("Settings saved successfully!");
        setCurrentPassword("");
        setNewPassword("");
        setConfirmPassword("");
      } else {
        const errorData = await response.text();
        setError(errorData || "Failed to save settings");
      }
    } catch (err) {
      setError("Failed to connect to server");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={`privacy-settings-container ${isDarkMode ? "dark-mode" : ""}`}>
      <h2 className="settings-title">Privacy Settings</h2>

      {message && <div className="settings-message success">{message}</div>}

      {error && <div className="settings-message error">{error}</div>}

      <div className="settings-group">
        <label className="settings-label">User Name</label>
        <input
          type="text"
          className="settings-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={loading}
        />
      </div>

      <div className="settings-group">
        <label className="settings-label">Current Password</label>
        <div className="password-input-container">
          <input
            type={showCurrentPassword ? "text" : "password"}
            className="settings-input"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            disabled={loading}
            placeholder="Enter current password to change"
          />
          <circle-button
            type="button"
            className="password-toggle-btn"
            onClick={() => setShowCurrentPassword(!showCurrentPassword)}
            disabled={loading}
          >
            {showCurrentPassword ? "👁️" : "👁️‍🗨️"}
          </circle-button>
        </div>
      </div>

      <div className="settings-group">
        <label className="settings-label">New Password</label>
        <div className="password-input-container">
          <input
            type={showNewPassword ? "text" : "password"}
            className="settings-input"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            disabled={loading}
            placeholder="Enter new password (min 6 characters)"
          />
          <circle-button
            type="button"
            className="password-toggle-btn"
            onClick={() => setShowNewPassword(!showNewPassword)}
            disabled={loading}
          >
            {showNewPassword ? "👁️" : "👁️‍🗨️"}
          </circle-button>
        </div>
      </div>

      <div className="settings-group">
        <label className="settings-label">Confirm New Password</label>
        <div className="password-input-container">
          <input
            type={showConfirmPassword ? "text" : "password"}
            className="settings-input"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            disabled={loading}
            placeholder="Confirm new password"
          />
          <circle-button
            type="button"
            className="password-toggle-btn"
            onClick={() => setShowConfirmPassword(!showConfirmPassword)}
            disabled={loading}
          >
            {showConfirmPassword ? "👁️" : "👁️‍🗨️"}
          </circle-button>
        </div>
      </div>

      <div className="settings-actions">
        <button
          className="save-button"
          onClick={handleSave}
          disabled={loading}
        >
          {loading ? "Saving..." : "Save Changes"}
        </button>
        <button
          className="back-button"
          onClick={() => navigate(-1)}
          disabled={loading}
        >
          ⬅ Back
        </button>
      </div>
    </div>
  );
};

export default PrivacySettings;
