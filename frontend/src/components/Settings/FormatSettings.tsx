import React from 'react';
import { ResponseFormat } from '../SettingsModal';

interface FormatSettingsProps {
  displayFormat: ResponseFormat;
  displaySchema: string;
  isExistingConversation: boolean;
  onFormatChange: (format: ResponseFormat) => void;
  onSchemaChange: (schema: string) => void;
  styles: any;
}

export const FormatSettings: React.FC<FormatSettingsProps> = ({
  displayFormat,
  displaySchema,
  isExistingConversation,
  onFormatChange,
  onSchemaChange,
  styles,
}) => {
  return (
    <>
      {/* Locked Configuration Info */}
      {isExistingConversation && (
        <div style={styles.infoBox}>
          <p style={styles.infoText}>
            <strong>🔒 Locked Configuration</strong>
          </p>
          <p style={styles.infoText}>
            This conversation is using <strong>{displayFormat.toUpperCase()}</strong> format.
            The response format cannot be changed after a conversation has started.
          </p>
        </div>
      )}

      {/* Only show radio buttons for new conversations */}
      {!isExistingConversation && (
        <>
          <label style={styles.label}>
            Response Format
            <p style={styles.description}>
              Choose how the AI should format its responses.
            </p>
          </label>
          <div style={styles.radioGroup}>
            <label style={styles.radioLabel}>
              <input
                type="radio"
                name="responseFormat"
                value="text"
                checked={displayFormat === 'text'}
                onChange={(e) => onFormatChange(e.target.value as ResponseFormat)}
                style={styles.radio}
              />
              <span>Plain Text (Default)</span>
            </label>
            <label style={styles.radioLabel}>
              <input
                type="radio"
                name="responseFormat"
                value="json"
                checked={displayFormat === 'json'}
                onChange={(e) => onFormatChange(e.target.value as ResponseFormat)}
                style={styles.radio}
              />
              <span>JSON</span>
            </label>
            <label style={styles.radioLabel}>
              <input
                type="radio"
                name="responseFormat"
                value="xml"
                checked={displayFormat === 'xml'}
                onChange={(e) => onFormatChange(e.target.value as ResponseFormat)}
                style={styles.radio}
              />
              <span>XML</span>
            </label>
          </div>
        </>
      )}

      {/* Schema Display/Input for JSON/XML */}
      {(displayFormat === 'json' || displayFormat === 'xml') && (
        <div style={styles.schemaSection}>
          <label style={styles.label}>
            Response Schema {isExistingConversation ? '' : '(Required)'}
            <p style={styles.description}>
              {isExistingConversation
                ? `Schema for this ${displayFormat.toUpperCase()} conversation:`
                : `Define the structure for the ${displayFormat.toUpperCase()} response. This schema will be used to instruct the AI on the exact format to follow.`}
            </p>
          </label>
          <textarea
            value={displaySchema}
            onChange={(e) => onSchemaChange(e.target.value)}
            placeholder={`Enter ${displayFormat.toUpperCase()} schema example...`}
            style={{
              ...styles.textarea,
              ...(isExistingConversation ? styles.textareaReadonly : {}),
            }}
            disabled={isExistingConversation}
            readOnly={isExistingConversation}
          />
        </div>
      )}
    </>
  );
};
