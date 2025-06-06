import React, { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import "./Analysis.css"; // Make sure this file includes the CSS you shared

const Analysis = () => {
  const { id: entryId } = useParams();
  const token = localStorage.getItem("token");
  const navigate = useNavigate();

  const [entry, setEntry] = useState(null);
  const [mood, setMood] = useState(null);
  const [error, setError] = useState("");

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
        const moodRes = await fetch(
          `http://localhost:8080/api/entries/${entryId}/mood`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          },
        );

        if (!moodRes.ok) {
          const text = await moodRes.text();
          throw new Error(`Mood fetch failed: ${text}`);
        }

        const moodData = await moodRes.json();
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
      <div className="content-box">
        <h2>Error: {error}</h2>
      </div>
    );
  }
  if (!entry || !mood) {
    return (
      <div className="content-box">
        <h2>Loading...</h2>
      </div>
    );
  }

  return (
    <div className="content-box">
      <div className="inner-container">
        <div className="entry-section">
          <h2>Journal Entry</h2>
          <h3>{entry.title}</h3>
          <p className="entry-content">{entry.text}</p>
        </div>

        <div className="analysis-section">
          <h2>Mood Analysis</h2>
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
            <strong>Suggestions:</strong> {mood.suggestions}
          </p>

          {mood.emotions.length > 0 && (
            <>
              <h3>Detected Emotions</h3>
              <ul>
                {mood.emotions.map((e, i) => (
                  <li key={i}>{e.label}: {(e.score * 100).toFixed(1)}%</li>
                ))}
              </ul>
            </>
          )}

          <p>
            <small>
              Analyzed: {new Date(mood.analyzed_at).toLocaleString()}
            </small>
          </p>

          <button className="back-button" onClick={() => navigate("/Homepage")}>
            ← Back to Homepage
          </button>
        </div>
      </div>
    </div>
  );
};

export default Analysis;
