// Policy.js
import React from "react";
import { Link } from "react-router-dom";
import "./Policy.css"; // Optional: style as needed

const Policy = () => {
  return (
    <div className="policy-container">
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

