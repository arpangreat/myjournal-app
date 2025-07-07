// Policy.js
import React, { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import "./Policy.css"; // Optional: style as needed

const Policy = () => {

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
  
  return (
    <div className={`policy-container ${isDarkMode ? "dark-mode" : ""}`}>
      <h1>Terms & Policies</h1>
      <h4>
        Welcome to our Terms and Policies page. By using this app, you agree to
        the following terms:
      </h4>
      <p>Your journal data is private and stored securely.
         You are responsible for safeguarding your login credentials.
         We do not sell your personal data to third parties.
         By continuing, you agree to our use of cookies and analytics.
         Any inappropriate use of the app may result in account suspension.
        </p>

      <p>These terms may be updated periodically. Please check back often.</p>
      <Link to="/Homepage" className="back-link">← Back to Homepage</Link>
    </div>
  );
};

export default Policy;

