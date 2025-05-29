import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "./AddEntry.css";

const AddEntry = ({ onAddEntry }) => {
  const [title, setTitle] = useState("");
  const [text, setText] = useState("");
  const navigate = useNavigate();

  const handleSubmit = () => {
    if (title.trim() && text.trim()) {
      const newEntry = {
        date: new Date().toLocaleDateString(),
        title,
        text,
      };
      if (onAddEntry) onAddEntry(newEntry);
      navigate("/Homepage"); // navigate back to homepage
    } else {
      alert("Please fill in both fields");
    }
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
      />
      <textarea
        placeholder="What's on your mind?"
        value={text}
        onChange={(e) => setText(e.target.value)}
        className="entry-text"
      />
      <button onClick={handleSubmit} className="submit-entry-btn">
        ➕ Add Entry
      </button>
    </div>
  );
};

export default AddEntry;

