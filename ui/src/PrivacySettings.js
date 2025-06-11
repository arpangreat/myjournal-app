// PrivacySettings.js
import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "./PrivacySettings.css"; // Optional: for styles


const PrivacySettings = () => {
  const navigate = useNavigate();
  const [name, setName] = useState(""); // Replace with actual data source
  const [dob, setDob] = useState("dd/mm/yyyy"); // Replace with actual data source
  const [isPrivate, setIsPrivate] = useState(true);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const handleSave = () => {
    // Implement save logic or API call
    console.log("Settings saved.");
  };

  return (
    <div className="privacy-settings-container" >
      <h2 className="settings-title">Privacy Settings</h2>

      <div className="settings-group">
        <label className="settings-label">Full Name</label>
        <input
          type="text"
          className="settings-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>

      <div className="settings-group">
        <label className="settings-label">Date of Birth</label>
        <input
          type="date"
          className="settings-input"
          value={dob}
          onChange={(e) => setDob(e.target.value)}
        />
      </div>

      <div className="settings-group">
        <label className="settings-label">Current Password</label>
        <input
          type="password"
          className="settings-input"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
        />
      </div>

      <div className="settings-group">
        <label className="settings-label">New Password</label>
        <input
          type="password"
          className="settings-input"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
        />
      </div>

      <div className="settings-group">
        <label className="settings-label">Confirm New Password</label>
        <input
          type="password"
          className="settings-input"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
        />
      </div>

      <button className="save-button" onClick={handleSave}>
        Save Changes
      </button>
       <button onClick={() => navigate(-1)}>⬅ Back</button>
    </div>
  );
};

export default PrivacySettings;