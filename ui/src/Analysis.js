import React, { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import "./Analysis.css";

const Analysis = () => {
  const { id: entryId } = useParams();
  const token = localStorage.getItem("token");
  const navigate = useNavigate();

  const [entry, setEntry] = useState(null);
  const [mood, setMood] = useState(null);
  const [error, setError] = useState("");

  // Add dark mode state that reads from localStorage
  const [isDarkMode, setIsDarkMode] = useState(() => {
    const savedDarkMode = localStorage.getItem("darkMode");
    return savedDarkMode === "true";
  });

  // Listen for dark mode changes from localStorage (cross-tab sync)
  useEffect(() => {
    const handleStorageChange = (e) => {
      if (e.key === "darkMode") {
        setIsDarkMode(e.newValue === "true");
      }
    };

    window.addEventListener("storage", handleStorageChange);

    // Also check for changes periodically (in case user changes mode in same tab)
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

  // Apply dark mode styles to document body
  useEffect(() => {
    if (isDarkMode) {
      document.body.classList.add("dark-mode");
    } else {
      document.body.classList.remove("dark-mode");
    }

    // Cleanup when component unmounts
    return () => {
      document.body.classList.remove("dark-mode");
    };
  }, [isDarkMode]);

  useEffect(() => {
    if (!token) {
      navigate("/login");
      return;
    }

    const fetchEntryAndMood = async () => {
      try {
        // Fetch all entries and find the one with the given ID
        const entryRes = await fetch("http://localhost:8080/api/entries", {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });

        const allEntries = await entryRes.json();
        const matchedEntry = allEntries.find((e) =>
          e.id.toString() === entryId
        );

        if (!matchedEntry) {
          throw new Error("Entry not found.");
        }

        setEntry(matchedEntry);

        // Fetch mood analysis
        // Retry fetching mood analysis up to 10 times
        let moodData = null;
        let attempts = 0;
        while (attempts < 10) {
          const moodRes = await fetch(
            `http://localhost:8080/api/entries/${entryId}/mood`,
            {
              headers: {
                Authorization: `Bearer ${token}`,
              },
            },
          );

          if (moodRes.ok) {
            moodData = await moodRes.json();
            break;
          }

          // Wait 2 seconds before retrying
          await new Promise((res) => setTimeout(res, 2000));
          attempts++;
        }

        if (!moodData) {
          throw new Error(
            "Mood analysis is still processing. Please try again later.",
          );
        }

        setMood(moodData);
      } catch (err) {
        console.error("Analysis error:", err);
        setError(err.message);
      }
    };

    fetchEntryAndMood();
  }, [entryId, token, navigate]);

  if (error) {
    return (
      <div className={`main-container ${isDarkMode ? "dark-mode" : ""}`}>
        <div className="content-box">
          <h2>Error: {error}</h2>
        </div>
      </div>
    );
  }
  if (!entry) {
    return (
      <div className="main-container">
        <div className="content-box">
          <div className="spinner" />
          <p className="loading-text">Loading journal entry...</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`main-container ${isDarkMode ? "dark-mode" : ""}`}>
      <div className="content-box">
        <div className="inner-container">
          <div className="entry-section">
            <h2>Journal Entry</h2>
            <div className="entry-scrollable-container">
              <h3 className="entry-title">{entry.title}</h3>
              <div className="entry-content-container">
                <p className="entry-content">{entry.text}</p>
              </div>
            </div>
          </div>
          <div className="analysis-section">
            <h2>Mood Analysis</h2>
            {!mood
              ? (
                <div className="analysis-loading-container">
                  <div className="spinner" />
                  <p className="loading-text">Analyzing your mood...</p>
                </div>
              )
              : (
                <div className="analysis-content-container">
                  <p>
                    <strong>Sentiment:</strong> {mood.overall_sentiment}
                  </p>
                  <p>
                    <strong>Score:</strong> {mood.sentiment_score.toFixed(2)}
                  </p>
                  <p>
                    <strong>Summary:</strong> {mood.summary}
                  </p>
                  <p>
                    <strong>Suggestions:</strong>{" "}
                    <span className="typed-suggestion">{mood.suggestions}</span>
                  </p>

                  {mood.emotions.length > 0 && (
                    <>
                      <h3>Detected Emotions</h3>
                      <ul>
                        {mood.emotions.map((e, i) => (
                          <li key={i}>
                            {e.label}: {(e.score * 100).toFixed(1)}%
                          </li>
                        ))}
                      </ul>
                    </>
                  )}

                  <p>
                    <small>
                      Analyzed: {new Date(mood.analyzed_at).toLocaleString()}
                    </small>
                  </p>
                </div>
              )}

            <button
              className="back-button"
              onClick={() => navigate("/Homepage")}
            >
              ← Back to Homepage
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Analysis;
