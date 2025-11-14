import React from 'react';
import { ProviderType } from '../SettingsModal';

interface ProviderSelectorProps {
  provider: ProviderType;
  onProviderChange: (provider: ProviderType) => void;
  styles: any;
}

export const ProviderSelector: React.FC<ProviderSelectorProps> = ({
  provider,
  onProviderChange,
  styles,
}) => {
  return (
    <div style={styles.providerSection}>
      <label style={styles.label}>
        LLM Provider
        <p style={styles.description}>
          Choose between direct OpenRouter API or Genkit framework.
        </p>
      </label>
      <div style={styles.providerToggle}>
        <label style={{
          ...styles.providerOption,
          ...(provider === 'openrouter' ? styles.providerOptionActive : {}),
        }}>
          <input
            type="radio"
            name="provider"
            value="openrouter"
            checked={provider === 'openrouter'}
            onChange={(e) => onProviderChange(e.target.value as ProviderType)}
            style={styles.radio}
          />
          <span>OpenRouter (Direct API)</span>
        </label>
        <label style={{
          ...styles.providerOption,
          ...(provider === 'genkit' ? styles.providerOptionActive : {}),
        }}>
          <input
            type="radio"
            name="provider"
            value="genkit"
            checked={provider === 'genkit'}
            onChange={(e) => onProviderChange(e.target.value as ProviderType)}
            style={styles.radio}
          />
          <span>Genkit (Firebase Framework)</span>
        </label>
      </div>
    </div>
  );
};
