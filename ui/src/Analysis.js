import { useJournal } from "./context/JournalContext";
import { useNavigate } from "react-router-dom";
import { useState } from "react";
import "./Analysis.css";

const Analysis = () => {
    const { selectedEntry } = useJournal(); 
    console.log("Received Entry in Analysis:", JSON.stringify(selectedEntry, null, 2));


    const navigate = useNavigate();

    const [showAnalysis, setShowAnalysis] = useState(false);

    return (
        <div className="content-box">
            <div className="inner-container">
            
                <div className="entry-section">
                    {selectedEntry ? (
                        <div>
                            <h2>{selectedEntry.title}</h2>
                            <p className="entry-content">{selectedEntry.content}</p>
                        </div>
                    ) : (
                        <p>No entry selected.</p>
                    )}

                    <button className="back-button" onClick={() => navigate("/Homepage")}>
                        Back 
                    </button>
                </div>
            
                <div className="analysis-section">
                    <h3 onClick={() => setShowAnalysis(!showAnalysis)}>
                        Analysis {showAnalysis ? "(Hide)" : "(Show)"}
                    </h3>
                    {showAnalysis && <p>Coming Soon: Mood Analysis & Keyword Extraction!</p>}
                </div>    
            </div>
        </div>
    );
};

export default Analysis;
