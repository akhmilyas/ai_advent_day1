import React from 'react';

interface PromptSettingsProps {
  systemPrompt: string;
  tempPrompt: string;
  onPromptChange: (prompt: string) => void;
  styles: any;
}

export const PromptSettings: React.FC<PromptSettingsProps> = ({
  systemPrompt,
  tempPrompt,
  onPromptChange,
  styles,
}) => {
  return (
    <>
      <label style={styles.label}>
        System Prompt
        <p style={styles.description}>
          This prompt will be combined with the default system prompt to guide the AI's behavior.
        </p>
      </label>

      {systemPrompt && (
        <div style={styles.currentPromptSection}>
          <p style={styles.currentPromptLabel}>Current System Prompt:</p>
          <div style={styles.currentPromptDisplay}>
            {systemPrompt}
          </div>
        </div>
      )}

      <label style={styles.editLabel}>
        {systemPrompt ? 'Edit System Prompt' : 'Enter System Prompt'}
      </label>
      <textarea
        value={tempPrompt}
        onChange={(e) => onPromptChange(e.target.value)}
        placeholder="Enter your custom system prompt..."
        style={styles.textarea}
      />
    </>
  );
};
