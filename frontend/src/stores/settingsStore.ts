import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ResponseFormat = 'text' | 'json' | 'xml';
export type ProviderType = 'openrouter' | 'genkit';

interface SettingsState {
  // State
  selectedModel: string;
  temperature: number;
  systemPrompt: string;
  responseFormat: ResponseFormat;
  responseSchema: string;
  provider: ProviderType;
  useWarAndPeace: boolean;
  warAndPeacePercent: number;
  settingsOpen: boolean;

  // Actions
  setModel: (model: string) => void;
  setTemperature: (temp: number) => void;
  setSystemPrompt: (prompt: string) => void;
  setResponseFormat: (format: ResponseFormat) => void;
  setResponseSchema: (schema: string) => void;
  setProvider: (provider: ProviderType) => void;
  setUseWarAndPeace: (use: boolean) => void;
  setWarAndPeacePercent: (percent: number) => void;
  setSettingsOpen: (open: boolean) => void;
  reset: () => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      // Initial state
      selectedModel: '',
      temperature: 0.7,
      systemPrompt: '',
      responseFormat: 'text',
      responseSchema: '',
      provider: 'openrouter',
      useWarAndPeace: false,
      warAndPeacePercent: 100,
      settingsOpen: false,

      // Actions
      setModel: (model) => set({ selectedModel: model }),
      setTemperature: (temp) => set({ temperature: temp }),
      setSystemPrompt: (prompt) => set({ systemPrompt: prompt }),
      setResponseFormat: (format) => set({ responseFormat: format }),
      setResponseSchema: (schema) => set({ responseSchema: schema }),
      setProvider: (provider) => set({ provider }),
      setUseWarAndPeace: (use) => set({ useWarAndPeace: use }),
      setWarAndPeacePercent: (percent) => set({ warAndPeacePercent: percent }),
      setSettingsOpen: (open) => set({ settingsOpen: open }),

      reset: () => set({
        selectedModel: '',
        temperature: 0.7,
        systemPrompt: '',
        responseFormat: 'text',
        responseSchema: '',
        provider: 'openrouter',
        useWarAndPeace: false,
        warAndPeacePercent: 100,
        settingsOpen: false
      })
    }),
    {
      name: 'settings-storage',
      // Only persist these fields
      partialize: (state) => ({
        selectedModel: state.selectedModel,
        temperature: state.temperature,
        systemPrompt: state.systemPrompt,
        responseFormat: state.responseFormat,
        responseSchema: state.responseSchema,
        provider: state.provider,
        useWarAndPeace: state.useWarAndPeace,
        warAndPeacePercent: state.warAndPeacePercent
        // Don't persist settingsOpen
      })
    }
  )
);
