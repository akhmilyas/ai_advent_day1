import React, { useState } from 'react';

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

  const handleSave = () => {
    onSystemPromptChange(tempPrompt);
    onClose();
  };

  const handleCancel = () => {
    setTempPrompt(systemPrompt);
    onClose();
  };

  if (!isOpen) return null;

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
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            style={styles.saveButton}
          >
            Save
          </button>
        </div>
      </div>
    </>
  );
};

const styles = {
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
    backgroundColor: '#ffffff',
    borderRadius: '8px',
    boxShadow: '0 4px 16px rgba(0, 0, 0, 0.15)',
    zIndex: 1000,
    minWidth: '500px',
    maxWidth: '600px',
    maxHeight: '80vh',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    borderBottom: '1px solid #e0e0e0',
  },
  title: {
    margin: 0,
    fontSize: '20px',
    fontWeight: 'bold',
  },
  closeButton: {
    background: 'none',
    border: 'none',
    fontSize: '24px',
    cursor: 'pointer',
    color: '#666',
    padding: 0,
    width: '32px',
    height: '32px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
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
  },
  description: {
    margin: '8px 0 0 0',
    fontSize: '12px',
    color: '#666',
    fontWeight: 'normal',
  },
  textarea: {
    width: '100%',
    minHeight: '150px',
    padding: '12px',
    fontSize: '14px',
    fontFamily: 'monospace',
    border: '1px solid #ccc',
    borderRadius: '4px',
    boxSizing: 'border-box' as const,
    resize: 'vertical' as const,
  },
  footer: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: '10px',
    padding: '20px',
    borderTop: '1px solid #e0e0e0',
    backgroundColor: '#f9f9f9',
  },
  cancelButton: {
    padding: '8px 16px',
    border: '1px solid #ccc',
    borderRadius: '4px',
    backgroundColor: '#fff',
    cursor: 'pointer',
    fontSize: '14px',
    transition: 'background-color 0.3s ease',
  },
  saveButton: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '4px',
    backgroundColor: '#007bff',
    color: '#fff',
    cursor: 'pointer',
    fontSize: '14px',
    transition: 'background-color 0.3s ease',
  },
};
