import React from 'react';
import { Model } from '../../services/chat';

interface ModelSettingsProps {
  selectedModel: string;
  availableModels: Model[];
  temperature: number;
  onModelChange: (model: string) => void;
  onTemperatureChange: (temperature: number) => void;
  styles: any;
}

export const ModelSettings: React.FC<ModelSettingsProps> = ({
  selectedModel,
  availableModels,
  temperature,
  onModelChange,
  onTemperatureChange,
  styles,
}) => {
  return (
    <>
      {/* Model Selector */}
      <div style={styles.modelSection}>
        <label style={styles.label}>
          AI Model
          <p style={styles.description}>
            Select which AI model to use for generating responses.
          </p>
        </label>
        <select
          value={selectedModel}
          onChange={(e) => onModelChange(e.target.value)}
          style={styles.select}
        >
          {availableModels.length === 0 && (
            <option value="">Loading models...</option>
          )}
          {availableModels.map((model) => (
            <option key={model.id} value={model.id}>
              {model.name} ({model.provider})
            </option>
          ))}
        </select>
      </div>

      {/* Temperature Slider */}
      <div style={styles.temperatureSection}>
        <label style={styles.label}>
          Temperature: {temperature.toFixed(2)}
          <p style={styles.description}>
            Controls randomness: Lower values (0.0-0.5) = more focused and deterministic, Higher values (0.5-2.0) = more creative and random.
          </p>
        </label>
        <input
          type="range"
          min="0"
          max="2"
          step="0.01"
          value={temperature}
          onChange={(e) => onTemperatureChange(parseFloat(e.target.value))}
          style={styles.slider}
        />
        <div style={styles.sliderLabels}>
          <span style={styles.sliderLabel}>0.0 (Focused)</span>
          <span style={styles.sliderLabel}>1.0 (Balanced)</span>
          <span style={styles.sliderLabel}>2.0 (Creative)</span>
        </div>
      </div>
    </>
  );
};
