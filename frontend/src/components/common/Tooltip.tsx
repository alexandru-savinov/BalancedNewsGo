import React, { useState, ReactNode } from 'react';
import styles from './Tooltip.module.css'; // We'll create this CSS module next

interface TooltipProps {
  children: ReactNode; // The element that triggers the tooltip on hover
  text: string | ReactNode; // The content of the tooltip
  position?: 'top' | 'bottom' | 'left' | 'right'; // Optional position
}

const Tooltip: React.FC<TooltipProps> = ({ children, text, position = 'top' }) => {
  const [isVisible, setIsVisible] = useState(false);

  const showTooltip = () => setIsVisible(true);
  const hideTooltip = () => setIsVisible(false);

  return (
    <div
      className={styles.tooltipContainer}
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip} // Added for accessibility
      onBlur={hideTooltip}  // Added for accessibility
      tabIndex={0} // Make it focusable if the child isn't naturally
    >
      {children}
      {isVisible && (
        <div className={`${styles.tooltipText} ${styles[position]}`}>
          {text}
          <span className={styles.tooltipArrow} /> {/* Optional arrow */}
        </div>
      )}
    </div>
  );
};

export default Tooltip;