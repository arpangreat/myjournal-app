// src/About.js
import React from 'react';
import { useNavigate } from "react-router-dom";
import './About.css';

const About = () => {
  const navigate = useNavigate();
  return (
    <div className="about-container">
      <h1>About This Journal App</h1>
      <p>
        Welcome to your personal Journal App – a secure and intuitive platform designed to help you record your thoughts, track your goals, and reflect on your daily experiences.
      </p>
      <p>
        This app offers features like custom journal entries, dark mode, personalized themes, habit tracking, and much more, all tailored for self-improvement and mental clarity.
      </p>
      <p>
        Your data is private and remains stored only on your device, unless you choose to back it up externally.
      </p>
      <button onClick={() => navigate(-1)}>⬅ Back</button>
    </div>
  );
};

export default About;