import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';
import { ChatService, Model } from '../services/chat';

export type ResponseFormat = 'text' | 'json' | 'xml';
export type ProviderType = 'openrouter' | 'genkit';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  systemPrompt: string;
  onSystemPromptChange: (prompt: string) => void;
  responseFormat: ResponseFormat;
  onResponseFormatChange: (format: ResponseFormat) => void;
  responseSchema: string;
  onResponseSchemaChange: (schema: string) => void;
  conversationFormat?: ResponseFormat | null;
  conversationSchema?: string;
  isExistingConversation: boolean;
  selectedModel: string;
  onModelChange: (model: string) => void;
  temperature: number;
  onTemperatureChange: (temperature: number) => void;
  provider: ProviderType;
  onProviderChange: (provider: ProviderType) => void;
  useWarAndPeace: boolean;
  onUseWarAndPeaceChange: (use: boolean) => void;
  warAndPeacePercent: number;
  onWarAndPeacePercentChange: (percent: number) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  systemPrompt,
  onSystemPromptChange,
  responseFormat,
  onResponseFormatChange,
  responseSchema,
  onResponseSchemaChange,
  conversationFormat,
  conversationSchema,
  isExistingConversation,
  selectedModel,
  onModelChange,
  temperature,
  onTemperatureChange,
  provider,
  onProviderChange,
  useWarAndPeace,
  onUseWarAndPeaceChange,
  warAndPeacePercent,
  onWarAndPeacePercentChange,
}) => {
  // Initialize with the correct format from the start
  const initialFormat = conversationFormat || responseFormat;
  const initialSchema = (conversationSchema !== undefined && conversationSchema !== '') ? conversationSchema : responseSchema;

  const [tempPrompt, setTempPrompt] = useState(systemPrompt);
  const [tempFormat, setTempFormat] = useState<ResponseFormat>(initialFormat);
  const [tempSchema, setTempSchema] = useState(initialSchema);
  const [tempModel, setTempModel] = useState(selectedModel);
  const [tempTemperature, setTempTemperature] = useState(temperature);
  const [tempProvider, setTempProvider] = useState<ProviderType>(provider);
  const [tempWarAndPeacePercent, setTempWarAndPeacePercent] = useState(warAndPeacePercent);
  const [availableModels, setAvailableModels] = useState<Model[]>([]);
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');
  const chatService = new ChatService();

  // Fetch available models on mount
  useEffect(() => {
    const fetchModels = async () => {
      try {
        const models = await chatService.getAvailableModels();
        setAvailableModels(models);
      } catch (error) {
        console.error('[SettingsModal] Failed to fetch models:', error);
      }
    };
    fetchModels();
  }, []);

  // Update state when modal opens or conversation changes
  React.useEffect(() => {
    console.log('[SettingsModal] useEffect triggered, isOpen:', isOpen, 'conversationFormat:', conversationFormat);

    setTempPrompt(systemPrompt);
    setTempModel(selectedModel);
    setTempTemperature(temperature);
    setTempProvider(provider);
    setTempWarAndPeacePercent(warAndPeacePercent);
    // For existing conversations with a format, use the locked format
    // Otherwise use the user's preference from localStorage
    if (conversationFormat) {
      console.log('[SettingsModal] Setting tempFormat to conversationFormat:', conversationFormat);
      setTempFormat(conversationFormat);
    } else {
      console.log('[SettingsModal] Setting tempFormat to responseFormat:', responseFormat);
      setTempFormat(responseFormat);
    }

    if (conversationSchema !== undefined && conversationSchema !== '') {
      setTempSchema(conversationSchema);
    } else {
      setTempSchema(responseSchema);
    }
  }, [systemPrompt, responseFormat, responseSchema, conversationFormat, conversationSchema, selectedModel, temperature, provider, warAndPeacePercent]);

  // For display, always use tempFormat (which is set from conversation or user preference)
  const displayFormat = tempFormat;
  const displaySchema = tempSchema;

  console.log('[SettingsModal] Render with:', { displayFormat, tempFormat, conversationFormat, isExistingConversation });

  const handleSave = () => {
    onSystemPromptChange(tempPrompt);
    onModelChange(tempModel);
    onTemperatureChange(tempTemperature);
    onProviderChange(tempProvider);
    onWarAndPeacePercentChange(tempWarAndPeacePercent);
    // Only save format changes if it's a new conversation
    if (!isExistingConversation) {
      onResponseFormatChange(tempFormat);
      onResponseSchemaChange(tempSchema);
    }
    onClose();
  };

  const handleCancel = () => {
    setTempPrompt(systemPrompt);
    setTempFormat(responseFormat);
    setTempSchema(responseSchema);
    setTempModel(selectedModel);
    setTempTemperature(temperature);
    setTempProvider(provider);
    setTempWarAndPeacePercent(warAndPeacePercent);
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
          {/* Provider Selector */}
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
                ...(tempProvider === 'openrouter' ? styles.providerOptionActive : {}),
              }}>
                <input
                  type="radio"
                  name="provider"
                  value="openrouter"
                  checked={tempProvider === 'openrouter'}
                  onChange={(e) => setTempProvider(e.target.value as ProviderType)}
                  style={styles.radio}
                />
                <span>OpenRouter (Direct API)</span>
              </label>
              <label style={{
                ...styles.providerOption,
                ...(tempProvider === 'genkit' ? styles.providerOptionActive : {}),
              }}>
                <input
                  type="radio"
                  name="provider"
                  value="genkit"
                  checked={tempProvider === 'genkit'}
                  onChange={(e) => setTempProvider(e.target.value as ProviderType)}
                  style={styles.radio}
                />
                <span>Genkit (Firebase Framework)</span>
              </label>
            </div>
          </div>

          {/* War and Peace Context Toggle */}
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
                  Context Size: {tempWarAndPeacePercent}%
                  <p style={styles.description}>
                    Controls what percentage of the War and Peace text to include (from the beginning).
                  </p>
                </label>
                <input
                  type="range"
                  min="1"
                  max="100"
                  step="1"
                  value={tempWarAndPeacePercent}
                  onChange={(e) => setTempWarAndPeacePercent(parseInt(e.target.value))}
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

          {/* Model Selector */}
          <div style={styles.modelSection}>
            <label style={styles.label}>
              AI Model
              <p style={styles.description}>
                Select which AI model to use for generating responses.
              </p>
            </label>
            <select
              value={tempModel}
              onChange={(e) => setTempModel(e.target.value)}
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
              Temperature: {tempTemperature.toFixed(2)}
              <p style={styles.description}>
                Controls randomness: Lower values (0.0-0.5) = more focused and deterministic, Higher values (0.5-2.0) = more creative and random.
              </p>
            </label>
            <input
              type="range"
              min="0"
              max="2"
              step="0.01"
              value={tempTemperature}
              onChange={(e) => setTempTemperature(parseFloat(e.target.value))}
              style={styles.slider}
            />
            <div style={styles.sliderLabels}>
              <span style={styles.sliderLabel}>0.0 (Focused)</span>
              <span style={styles.sliderLabel}>1.0 (Balanced)</span>
              <span style={styles.sliderLabel}>2.0 (Creative)</span>
            </div>
          </div>

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
                    onChange={(e) => setTempFormat(e.target.value as ResponseFormat)}
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
                    onChange={(e) => setTempFormat(e.target.value as ResponseFormat)}
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
                    onChange={(e) => setTempFormat(e.target.value as ResponseFormat)}
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
                onChange={(e) => setTempSchema(e.target.value)}
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

          {/* System Prompt (only for text format) */}
          {displayFormat === 'text' && (
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
                onChange={(e) => setTempPrompt(e.target.value)}
                placeholder="Enter your custom system prompt..."
                style={styles.textarea}
              />
            </>
          )}
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
    maxHeight: '85vh',
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
  textareaReadonly: {
    backgroundColor: colors.surfaceAlt,
    cursor: 'not-allowed',
    opacity: 0.9,
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
  radioGroup: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '10px',
    marginBottom: '20px',
  },
  radioLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: colors.text,
    fontSize: '14px',
    cursor: 'pointer',
    padding: '8px',
    borderRadius: '4px',
    transition: 'background-color 0.2s ease',
  },
  radio: {
    cursor: 'pointer',
    width: '16px',
    height: '16px',
  },
  schemaSection: {
    marginTop: '16px',
    marginBottom: '16px',
  },
  modelSection: {
    marginBottom: '20px',
  },
  select: {
    width: '100%',
    padding: '10px',
    fontSize: '14px',
    border: `1px solid ${colors.border}`,
    borderRadius: '4px',
    backgroundColor: colors.input,
    color: colors.text,
    cursor: 'pointer',
    boxSizing: 'border-box' as const,
    transition: 'background-color 0.3s ease, border-color 0.3s ease, color 0.3s ease',
  },
  infoBox: {
    backgroundColor: colors.surfaceAlt,
    border: `1px solid ${colors.border}`,
    borderRadius: '4px',
    padding: '12px',
    marginBottom: '16px',
  },
  infoText: {
    margin: '0 0 8px 0',
    fontSize: '13px',
    color: colors.text,
    lineHeight: '1.5',
  },
  schemaPreview: {
    marginTop: '12px',
    padding: '8px',
    backgroundColor: colors.input,
    borderRadius: '4px',
    border: `1px solid ${colors.border}`,
  },
  schemaPreviewLabel: {
    margin: '0 0 6px 0',
    fontSize: '11px',
    fontWeight: 'bold' as const,
    color: colors.textSecondary,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.5px',
  },
  schemaPreviewContent: {
    margin: 0,
    fontSize: '12px',
    color: colors.text,
    fontFamily: 'monospace',
    whiteSpace: 'pre-wrap' as const,
    wordBreak: 'break-word' as const,
    maxHeight: '200px',
    overflowY: 'auto' as const,
    lineHeight: '1.4',
  },
  temperatureSection: {
    marginBottom: '20px',
  },
  slider: {
    width: '100%',
    height: '6px',
    borderRadius: '3px',
    outline: 'none',
    cursor: 'pointer',
    marginBottom: '8px',
    accentColor: colors.buttonPrimary,
  },
  sliderLabels: {
    display: 'flex',
    justifyContent: 'space-between',
    paddingTop: '4px',
  },
  sliderLabel: {
    fontSize: '11px',
    color: colors.textSecondary,
  },
  providerSection: {
    marginBottom: '20px',
  },
  providerToggle: {
    display: 'flex',
    gap: '12px',
    flexDirection: 'row' as const,
  },
  providerOption: {
    flex: 1,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    padding: '12px',
    border: `2px solid ${colors.border}`,
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    backgroundColor: colors.surface,
    color: colors.text,
  },
  providerOptionActive: {
    borderColor: colors.buttonPrimary,
    backgroundColor: `${colors.buttonPrimary}15`,
  },
  warAndPeaceSection: {
    marginBottom: '20px',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: '12px',
    cursor: 'pointer',
  },
  checkbox: {
    marginTop: '2px',
    width: '18px',
    height: '18px',
    cursor: 'pointer',
  },
  checkboxText: {
    fontWeight: 500,
    color: colors.text,
  },
  percentageSliderSection: {
    marginTop: '16px',
    paddingLeft: '30px',
  },
});
