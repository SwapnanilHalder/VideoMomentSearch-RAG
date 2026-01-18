import { useState } from 'react';
import './App.css';

function App() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const formatTime = (seconds) => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  const handleSearch = async () => {
    if (!query.trim()) return;

    setLoading(true);
    setError("");
    setResults([]);

    try {
      const response = await fetch("http://localhost:8080/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: query })
      });

      if (!response.ok) {
        throw new Error("Failed to fetch results");
      }

      const data = await response.json();
      
      const formattedResults = data.map((item, index) => {
        const rawSeconds = item.start_seconds;
        
        const shouldAutoplay = index === 0 ? "1" : "0";

        return {
          ...item,
          displayTime: formatTime(rawSeconds),
          embedUrl: `https://www.youtube.com/embed/${item.video_id}?start=${rawSeconds}&autoplay=${shouldAutoplay}`
        };
      });

      setResults(formattedResults);

    } catch (err) {
      setError("Something went wrong. Is the backend running?");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') handleSearch();
  };

  return (
    <div className="container">
      <h1>🔍 Video Moment Search</h1>
      
      <div className="search-box">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search Query for Video Moment..."
          disabled={loading}
        />
        <button onClick={handleSearch} disabled={loading}>
          {loading ? "⏳..." : "Answer"}
        </button>
      </div>

      {error && <p className="error">{error}</p>}

      <div className="results-list">
        {results.map((result, index) => (
          <div key={index} className="result-card fade-in">
            
            <div className="result-header">
              <span className="rank-badge"># Search Result {index + 1}</span>
            </div>

            <div className="video-wrapper">
              <iframe
                src={result.embedUrl}
                title={`Result ${index + 1}`}
                frameBorder="0"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
              ></iframe>
            </div>

            <div className="info">
              <h2>{result.video_title}</h2>
              
              <span className="badge">Start Time: {result.displayTime}</span>
              <span className="score-badge">Confidence: {(result.score * 100).toFixed(1)}%</span>
              
              <div className="why-box">
                <h3>💡 Why this moment?</h3>
                <p>{result.explanation}</p>
              </div>

              <details>
                <summary>View Transcript Snippet</summary>
                <p className="snippet">"...{result.snippet}..."</p>
              </details>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;