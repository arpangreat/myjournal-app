// Policy.js
import React from "react";
import { Link } from "react-router-dom";
import "./Policy.css"; // Optional: style as needed

const Policy = () => {
  return (
    <div className="policy-container">
      <h1>Terms & Policies</h1>
      <p>Welcome to our Terms and Policies page. By using this app, you agree to the following terms:</p>
      <ul>
        <li>Your journal data is private and stored securely.</li>
        <li>You are responsible for safeguarding your login credentials.</li>
        <li>We do not sell your personal data to third parties.</li>
        <li>By continuing, you agree to our use of cookies and analytics.</li>
        <li>Any inappropriate use of the app may result in account suspension.</li>
      </ul>
      <p>These terms may be updated periodically. Please check back often.</p>
      <Link to="/" className="back-link">← Back to Homepage</Link>
    </div>
  );
};

export default Policy;