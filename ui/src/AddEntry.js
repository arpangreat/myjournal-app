import React, { useState } from "react";
import "./AddEntry.css";

const AddEntry = ({ onAddEntry }) => {
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");

  const handleSubmit = (e) => {
    e.preventDefault();
    if (title.trim() && content.trim()) {
      onAddEntry({
        title,
        content,
        date: new Date().toLocaleString(),
      });
      setTitle("");
      setContent("");
    }
  };

  return (
    <div className="add-entry-container">
      <h2>Add New Journal Entry</h2>
      <form className="entry-form" onSubmit={handleSubmit}>
        <input
          type="text"
          className="entry-title"
          placeholder="Entry Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <textarea
          className="entry-content"
          placeholder="Write your thoughts..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />
        <button type="submit" className="submit-button">Add Entry</button>
      </form>
    </div>
  );
};

export default AddEntry;