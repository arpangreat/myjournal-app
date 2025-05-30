import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "./AddEntry.css";

const AddEntry = ({ onAddEntry }) => {
  const [title, setTitle] = useState("");
  const [text, setText] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async () => {
    if (!title.trim() || !text.trim()) {
      alert("Please fill in both fields");
      return;
    }

    setIsSubmitting(true);

    try {
      // Get JWT token from localStorage
      const token = localStorage.getItem("token");

      if (!token) {
        alert("Please log in to add entries");
        navigate("/login");
        return;
      }

      const newEntry = {
        title: title.trim(),
        text: text.trim(),
        date: new Date().toLocaleDateString("en-US"), // MM/DD/YYYY format to match backend
      };

      const response = await fetch("http://localhost:8080/api/entries", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify(newEntry),
      });

      if (response.ok) {
        const savedEntry = await response.json();

        // Call the parent component's callback if provided
        // if (onAddEntry) {
        //   onAddEntry(savedEntry);
        // }

        // Clear the form
        setTitle("");
        setText("");

        // Navigate back to homepage
        navigate("/Homepage");
      } else if (response.status === 401) {
        // Token expired or invalid
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        alert("Session expired. Please log in again.");
        navigate("/login");
      } else {
        const errorText = await response.text();
        alert(`Failed to save entry: ${errorText}`);
      }
    } catch (error) {
      console.error("Error adding entry:", error);
      alert(
        "Failed to save entry. Please check your connection and try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate("/Homepage");
  };

  return (
    <div className="add-entry-page">
      <h2>📝 New Journal Entry</h2>

      <input
        type="text"
        placeholder="Entry Title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        className="entry-title"
        disabled={isSubmitting}
      />

      <textarea
        placeholder="What's on your mind?"
        value={text}
        onChange={(e) => setText(e.target.value)}
        className="entry-text"
        disabled={isSubmitting}
      />

      <div className="button-group">
        <button
          onClick={handleSubmit}
          className="submit-entry-btn"
          disabled={isSubmitting}
        >
          {isSubmitting ? "⏳ Saving..." : "➕ Add Entry"}
        </button>

        <button
          onClick={handleCancel}
          className="cancel-btn"
          disabled={isSubmitting}
        >
          ❌ Cancel
        </button>
      </div>
    </div>
  );
};

export default AddEntry;
