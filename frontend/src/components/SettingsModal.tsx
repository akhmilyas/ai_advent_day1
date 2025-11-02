import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  systemPrompt: string;
  onSystemPromptChange: (prompt: string) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  systemPrompt,
  onSystemPromptChange,
}) => {
  const [tempPrompt, setTempPrompt] = useState(systemPrompt);
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  const handleSave = () => {
    onSystemPromptChange(tempPrompt);
    onClose();
  };

  const handleCancel = () => {
    setTempPrompt(systemPrompt);
    onClose();
  };

  if (!isOpen) return null;

  const styles = getStyles(colors);

  return (
    <>
      {/* Backdrop */}
      <div
        style={styles.backdrop}
        onClick={handleCancel}
      />
      {/* Modal */}
      <div style={styles.modal}>
        <div style={styles.header}>
          <h2 style={styles.title}>Settings</h2>
          <button
            onClick={handleCancel}
            style={styles.closeButton}
            title="Close"
          >
            ✕
          </button>
        </div>

        <div style={styles.content}>
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
            onChange={(e) => setTempPrompt(e.target.value)}
            placeholder="Enter your custom system prompt..."
            style={styles.textarea}
          />
        </div>

        <div style={styles.footer}>
          <button
            onClick={handleCancel}
            style={styles.cancelButton}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = colors.border;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = colors.surface;
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            style={styles.saveButton}
            onMouseEnter={(e) => {
              e.currentTarget.style.opacity = '0.9';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.opacity = '1';
            }}
          >
            Save
          </button>
        </div>
      </div>
    </>
  );
};

const getStyles = (colors: ReturnType<typeof getTheme>) => ({
  backdrop: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    zIndex: 999,
  },
  modal: {
    position: 'fixed' as const,
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    backgroundColor: colors.surface,
    borderRadius: '8px',
    boxShadow: `0 4px 16px ${colors.shadow}`,
    zIndex: 1000,
    minWidth: '500px',
    maxWidth: '600px',
    maxHeight: '80vh',
    display: 'flex',
    flexDirection: 'column' as const,
    transition: 'background-color 0.3s ease, box-shadow 0.3s ease',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    borderBottom: `1px solid ${colors.border}`,
  },
  title: {
    margin: 0,
    fontSize: '20px',
    fontWeight: 'bold',
    color: colors.text,
  },
  closeButton: {
    background: 'none',
    border: 'none',
    fontSize: '24px',
    cursor: 'pointer',
    color: colors.textSecondary,
    padding: 0,
    width: '32px',
    height: '32px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    transition: 'color 0.3s ease',
  },
  content: {
    flex: 1,
    padding: '20px',
    overflowY: 'auto' as const,
  },
  label: {
    display: 'block',
    marginBottom: '12px',
    fontWeight: 'bold',
    color: colors.text,
  },
  description: {
    margin: '8px 0 0 0',
    fontSize: '12px',
    color: colors.textSecondary,
    fontWeight: 'normal',
  },
  currentPromptSection: {
    marginBottom: '16px',
    padding: '12px',
    backgroundColor: colors.surfaceAlt,
    borderRadius: '4px',
    border: `1px solid ${colors.border}`,
  },
  currentPromptLabel: {
    margin: '0 0 8px 0',
    fontSize: '12px',
    fontWeight: 'bold' as const,
    color: colors.textSecondary,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.5px',
  },
  currentPromptDisplay: {
    fontSize: '13px',
    lineHeight: '1.5',
    color: colors.text,
    whiteSpace: 'pre-wrap' as const,
    wordBreak: 'break-word' as const,
    maxHeight: '100px',
    overflowY: 'auto' as const,
    fontFamily: 'monospace',
  },
  editLabel: {
    display: 'block',
    marginTop: '12px',
    marginBottom: '8px',
    fontSize: '13px',
    fontWeight: 'bold' as const,
    color: colors.text,
  },
  textarea: {
    width: '100%',
    minHeight: '150px',
    padding: '12px',
    fontSize: '14px',
    fontFamily: 'monospace',
    border: `1px solid ${colors.border}`,
    borderRadius: '4px',
    boxSizing: 'border-box' as const,
    resize: 'vertical' as const,
    backgroundColor: colors.input,
    color: colors.text,
    boxShadow: 'none',
    transition: 'background-color 0.3s ease, border-color 0.3s ease, color 0.3s ease',
  },
  footer: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: '10px',
    padding: '20px',
    borderTop: `1px solid ${colors.border}`,
    backgroundColor: colors.surfaceAlt,
    transition: 'background-color 0.3s ease',
  },
  cancelButton: {
    padding: '8px 16px',
    border: `1px solid ${colors.border}`,
    borderRadius: '4px',
    backgroundColor: colors.surface,
    color: colors.text,
    cursor: 'pointer',
    fontSize: '14px',
    transition: 'background-color 0.3s ease, color 0.3s ease',
  },
  saveButton: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '4px',
    backgroundColor: colors.buttonPrimary,
    color: colors.buttonPrimaryText,
    cursor: 'pointer',
    fontSize: '14px',
    transition: 'opacity 0.3s ease',
  },
});
