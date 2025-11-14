import React from 'react';

interface WarAndPeaceSettingsProps {
  useWarAndPeace: boolean;
  warAndPeacePercent: number;
  onUseWarAndPeaceChange: (use: boolean) => void;
  onWarAndPeacePercentChange: (percent: number) => void;
  styles: any;
}

export const WarAndPeaceSettings: React.FC<WarAndPeaceSettingsProps> = ({
  useWarAndPeace,
  warAndPeacePercent,
  onUseWarAndPeaceChange,
  onWarAndPeacePercentChange,
  styles,
}) => {
  return (
    <div style={styles.warAndPeaceSection}>
      <label style={styles.checkboxLabel}>
        <input
          type="checkbox"
          checked={useWarAndPeace}
          onChange={(e) => onUseWarAndPeaceChange(e.target.checked)}
          style={styles.checkbox}
        />
        <div>
          <span style={styles.checkboxText}>Add War and Peace context</span>
          <p style={styles.description}>
            Appends the full text of "War and Peace" by Leo Tolstoy to the system prompt (3.2 MB).
            This provides extensive Russian literature context to the AI.
          </p>
        </div>
      </label>

      {/* Percentage Slider - only show when War and Peace is enabled */}
      {useWarAndPeace && (
        <div style={styles.percentageSliderSection}>
          <label style={styles.label}>
            Context Size: {warAndPeacePercent}%
            <p style={styles.description}>
              Controls what percentage of the War and Peace text to include (from the beginning).
            </p>
          </label>
          <input
            type="range"
            min="1"
            max="100"
            step="1"
            value={warAndPeacePercent}
            onChange={(e) => onWarAndPeacePercentChange(parseInt(e.target.value))}
            style={styles.slider}
          />
          <div style={styles.sliderLabels}>
            <span style={styles.sliderLabel}>1% (~32 KB)</span>
            <span style={styles.sliderLabel}>50% (~1.6 MB)</span>
            <span style={styles.sliderLabel}>100% (~3.2 MB)</span>
          </div>
        </div>
      )}
    </div>
  );
};
