import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import "./AddEntry.css";

const AddEntry = ({ onAddEntry }) => {
  const [title, setTitle] = useState("");
  const [text, setText] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Add dark mode state that reads from localStorage
  const [isDarkMode, setIsDarkMode] = useState(() => {
    const savedDarkMode = localStorage.getItem('darkMode');
    return savedDarkMode === 'true';
  });
  const navigate = useNavigate();
  
   // Font selection state
  const [selectedFont, setSelectedFont] = useState(() => {
    return localStorage.getItem('userFontPreference') || 'Arial';
  });
  const [showFontList, setShowFontList] = useState(false);

  const fonts = [
    "Arial",
    "Wide Latin",
    "Vladimir Script",
    "Showcard Gothic",
    "Algerian",
    "Bradley Hand ITC",
    "Matura MT Script Capitals",
    "Broadway",
    "Bauhaus 93",
    "Chiller",
    "Calibri",
    "Cambria",
    "Candara",
    "Comic Sans MS",
    "Consolas",
    "Constantia",
    "Corbel",
    "Courier New",
    "Franklin Gothic Medium",
    "Georgia",
    "Helvetica",
    "Impact",
    "Lucida Console",
    "Lucida Sans Unicode",
    "Palatino Linotype",
    "Segoe UI",
    "Tahoma",
    "Times New Roman",
    "Trebuchet MS",
    "Verdana",
    "Century Gothic",
    "Garamond",
    "Bookman Old Style",
    "Book Antiqua",
    "Elephant",
    "Futura",
    "Gill Sans MT",
    "Harlow Solid Italic",
    "Ink Free",
    "Kristen ITC",
    "Leelawadee UI",
    "Magneto",
    "MV Boli",
    "Perpetua",
    "Ravie",
    "Rockwell",
    "Showcard Gothic",
    "Snap ITC",
    "Stencil",
    "Tw Cen MT"
  ];

  // Listen for dark mode changes from localStorage (cross-tab sync)
  useEffect(() => {
    const handleStorageChange = (e) => {
      if (e.key === 'darkMode') {
        setIsDarkMode(e.newValue === 'true');
      }
    };

    window.addEventListener('storage', handleStorageChange);
    
    // Also check for changes periodically (in case user changes mode in same tab)
    const interval = setInterval(() => {
      const currentDarkMode = localStorage.getItem('darkMode') === 'true';
      if (currentDarkMode !== isDarkMode) {
        setIsDarkMode(currentDarkMode);
      }
    }, 1000);

    return () => {
      window.removeEventListener('storage', handleStorageChange);
      clearInterval(interval);
    };
  }, [isDarkMode]);

  // Apply dark mode styles to document body
  useEffect(() => {
    if (isDarkMode) {
      document.body.classList.add('dark-mode');
    } else {
      document.body.classList.remove('dark-mode');
    }
    
    // Cleanup when component unmounts
    return () => {
      document.body.classList.remove('dark-mode');
    };
  }, [isDarkMode]);

  // Function to handle font selection
  const handleFontSelect = (font) => {
    setSelectedFont(font);
    setShowFontList(false);
    // Save font preference to localStorage
    localStorage.setItem('userFontPreference', font);
  };

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
    <div className={`add-entry-page ${isDarkMode ? "dark-mode" : ""}`}>
      <h2>📝 New Journal Entry</h2>


      <div className="font-selector">
        <circle-button
          onClick={() => setShowFontList(!showFontList)}
          className="font-toggle-btn"
          type="button"
        >
          🎨
        </circle-button>
        {showFontList && (
          <ul className="font-list">
            {fonts.map((font) => (
              <li
                key={font}
                onClick={() => handleFontSelect(font)}
                style={{ 
                  fontFamily: font
                }}
                className={selectedFont === font ? 'selected-font' : ''}
              >
                {font}
              </li>
            ))}
          </ul>
        )}
      </div>


      <input
        type="text"
        placeholder="Entry Title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        className="entry-title"
        disabled={isSubmitting}
        style={{ fontFamily: selectedFont }}
      />

      <textarea
        placeholder="What's on your mind?"
        value={text}
        onChange={(e) => setText(e.target.value)}
        className="entry-text"
        disabled={isSubmitting}
        style={{ fontFamily: selectedFont }}
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
