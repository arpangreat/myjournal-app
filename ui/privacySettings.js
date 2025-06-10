import React, { useState } from "react";
import "./PrivacyStyles.css"; // Importing CSS file

const PrivacySettings = () => {
  const [name, setName] = useState("User Name");
  const [email, setEmail] = useState("user@example.com");
  const [appLock, setAppLock] = useState(false);

  const handleChangeName = () => {
    const newName = prompt("Enter your new display name:");
    if (newName) setName(newName);
  };

  const handleChangeEmail = () => {
    const newEmail = prompt("Enter your new email:");
    if (newEmail) setEmail(newEmail);
  };

  const handleDeleteAccount = () => {
    if (window.confirm("Are you sure you want to delete your account?")) {
      alert("Account deleted successfully.");
    }
  };

  return (
    <div className="settings-container">
      <h2>Privacy Settings</h2>
      <p>Name: {name}</p>
      <button className="button" onClick={handleChangeName}>Change Name</button>

      <p>Email: {email}</p>
      <button className="button" onClick={handleChangeEmail}>Change Email</button>

      <div className="toggle-container">
        <label>Enable App Lock:</label>
        <input
          type="checkbox"
          checked={appLock}
          onChange={() => setAppLock(!appLock)}
        />
      </div>

      <button className="delete-button" onClick={handleDeleteAccount}>
        Delete Account
      </button>
    </div>
  );
};


export default PrivacySettings;