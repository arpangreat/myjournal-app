// src/About.js
import React, { useEffect, useState } from 'react';
import { useNavigate } from "react-router-dom";
import './About.css';

const About = () => {

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

  const navigate = useNavigate();
  return (
    <div className={`about-container ${isDarkMode ? "dark-mode" : ""}`}>
      <h2>About This Journal App</h2>
      <p>
        Welcome to your personal Journal – our vision was to create a secure and intuitive platform designed to help users record their thoughts and reflect on daily experiences to improve themselves.
      </p>
      <p>
        It offers features like custom journal entries, dark mode, analysing your journal and gives you results to reflect on that, all tailored for self-improvement and mental clarity.
      </p>
      <p>
        Users data is private and remains stored only on their device, cause we have no idea what to do with it further.
      </p>
      <p>
        It is a personal and secure digital space designed to help users reflect,
        record, and revisit their thoughts and experiences. Whether you're logging daily events,
        tracking emotions, or preserving memories,  makes journaling intuitive, accessible,
        and meaningful.
      </p>
      <p> Obviously if users know how to write all their thoughts.</p>
      <p>We believe we should acknowledge everyones hard work and efforts and this Journal App is the result of the dedication and creativity of contributors:</p>
      <p> Swastik Acharyya</p>
      <p>Suhana Zaman</p>
      <p> Nilufa Sultana</p>
      <button onClick={() => navigate(-1)}>⬅ Back</button>
    </div>
  );
};

export default About;