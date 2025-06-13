import React, { useState, useEffect, useRef } from "react";
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

  // Ref for font selector container
  const fontSelectorRef = useRef(null);

  const fonts = [
    "Algerian",
    "Arial",
    "Book Antiqua",
    "Bookman Old Style",    
    "Bradley Hand ITC",
    "Broadway",
    "Calibri",
    "Cambria",
    "Candara",
    "Century Gothic",
    "Chiller",
    "Comic Sans MS",
    "Consolas",
    "Constantia",
    "Corbel",
    "Courier New",
    "Elephant",
    "Franklin Gothic Medium",
    "Futura",
    "Garamond",
    "Georgia",
    "Gill Sans MT",
    "Harlow Solid Italic",
    "Helvetica",
    "Impact",
    "Ink Free",
    "Kristen ITC",
    "Leelawadee UI",
    "Lucida Console",
    "Lucida Sans Unicode",
    "Magneto",
    "Matura MT Script Capitals", 
    "MV Boli",
    "Palatino Linotype",
    "Perpetua",
    "Ravie",
    "Rockwell",
    "Segoe UI",
    "Showcard Gothic",
    "Snap ITC",
    "Stencil",
    "Tahoma",
    "Times New Roman",
    "Trebuchet MS",
    "Tw Cen MT",
    "Vladimir Script",
    "Verdana",  
    "Wide Latin"
  ];

  // Handle clicks outside font selector
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (fontSelectorRef.current && !fontSelectorRef.current.contains(event.target)) {
        setShowFontList(false);
      }
    };

    if (showFontList) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showFontList]);

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


      <div className="font-selector" ref={fontSelectorRef}>
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
                  fontFamily: font,
                  color: selectedFont === font ? 'slategray' : 'inherit',
                  fontWeight: selectedFont === font ? 'bold' : 'normal'
                }}
                className={selectedFont === font ? 'selected-font' : ''}
              >
                {font}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="title-input-container">
        <div className="character-count" style={{ 
          textAlign: 'left', 
          fontSize: '12px'
        }}>
          {100 - title.length}
        </div>
        <input
          type="text"
          placeholder="Entry Title"
          value={title}
          onChange={(e) => {
            if (e.target.value.length <= 100) {
              setTitle(e.target.value)
            }
          }}
          className="entry-title"
          disabled={isSubmitting}
          style={{ fontFamily: selectedFont }}
          maxLength={100}
        />
      </div>

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
