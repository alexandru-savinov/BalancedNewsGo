import React from 'react';
import styles from './BiasSlider.module.css'; // We'll create this CSS module next

interface BiasSliderProps {
  score: number | null | undefined; // Assuming score ranges from -1 (left) to 1 (right)
  confidence?: number | null | undefined; // Optional confidence score
}

const BiasSlider: React.FC<BiasSliderProps> = ({ score, confidence }) => {
  // Handle null/undefined score - perhaps render nothing or a placeholder
  if (score === null || score === undefined) {
    return <div className={styles.sliderContainer}>Score N/A</div>;
  }

  // Normalize score from [-1, 1] to [0, 100] for positioning
  const normalizedScore = (score + 1) * 50;
  const positionPercent = Math.max(0, Math.min(100, normalizedScore)); // Clamp between 0 and 100

  // Basic color coding (can be refined)
  let trackColor = '#ccc'; // Default grey
  if (score < -0.2) trackColor = 'blue'; // Left-leaning
  if (score > 0.2) trackColor = 'red'; // Right-leaning

  return (
    <div className={styles.sliderContainer} title={`Score: ${score.toFixed(2)}${confidence ? `, Confidence: ${confidence.toFixed(2)}` : ''}`}>
      <div className={styles.sliderTrack} style={{ background: trackColor }}>
        <div
          className={styles.sliderThumb}
          style={{ left: `${positionPercent}%` }}
        />
      </div>
      <div className={styles.labels}>
        <span>Left</span>
        <span>Center</span>
        <span>Right</span>
      </div>
    </div>
  );
};

export default BiasSlider;