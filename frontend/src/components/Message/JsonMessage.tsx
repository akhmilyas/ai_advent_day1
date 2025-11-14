import React from 'react';
import { getTheme } from '../../themes';

interface JsonMessageProps {
  content: string;
  colors: ReturnType<typeof getTheme>;
}

export const JsonMessage: React.FC<JsonMessageProps> = ({ content, colors }) => {
  try {
    const parsed = JSON.parse(content);

    // Recursive function to render JSON nodes
    const renderNode = (value: any, key?: string, level: number = 0): React.ReactNode => {
      const indent = level * 20;

      // Handle null
      if (value === null) {
        return (
          <div key={key} style={{ marginLeft: `${indent}px`, padding: '4px 8px' }}>
            {key && <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>{key}: </span>}
            <span style={{ color: colors.textSecondary, fontFamily: 'monospace', fontStyle: 'italic' }}>null</span>
          </div>
        );
      }

      // Handle primitives (string, number, boolean)
      if (typeof value !== 'object') {
        return (
          <div key={key} style={{ marginLeft: `${indent}px`, padding: '4px 8px' }}>
            {key && <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>{key}: </span>}
            <span style={{
              color: typeof value === 'string' ? colors.text : colors.textSecondary,
              fontFamily: 'monospace'
            }}>
              {typeof value === 'string' ? `"${value}"` : String(value)}
            </span>
          </div>
        );
      }

      // Handle arrays
      if (Array.isArray(value)) {
        return (
          <div key={key} style={{ marginLeft: `${indent}px` }}>
            <div style={{
              padding: '6px 8px',
              backgroundColor: level % 2 === 0 ? colors.surfaceAlt : 'transparent',
              borderLeft: `3px solid ${colors.buttonPrimary}`,
              borderRadius: '4px',
              marginBottom: '4px',
            }}>
              {key && <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>{key}: </span>}
              <span style={{ color: colors.textSecondary, fontFamily: 'monospace' }}>[{value.length} items]</span>
            </div>
            <div>
              {value.map((item, idx) => (
                <React.Fragment key={idx}>
                  {renderNode(item, `[${idx}]`, level + 1)}
                </React.Fragment>
              ))}
            </div>
          </div>
        );
      }

      // Handle objects
      const entries = Object.entries(value);
      return (
        <div key={key} style={{ marginLeft: `${indent}px` }}>
          {key && (
            <div style={{
              padding: '6px 8px',
              backgroundColor: level % 2 === 0 ? colors.surfaceAlt : 'transparent',
              borderLeft: `3px solid ${colors.buttonPrimary}`,
              borderRadius: '4px',
              marginBottom: '4px',
            }}>
              <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>{key}</span>
              <span style={{ color: colors.textSecondary, fontFamily: 'monospace' }}> {'{'}...{'}'}</span>
            </div>
          )}
          <div>
            {entries.map(([k, v]) => (
              <React.Fragment key={k}>
                {renderNode(v, k, level + 1)}
              </React.Fragment>
            ))}
          </div>
        </div>
      );
    };

    return (
      <div>
        {/* Show original JSON */}
        <details style={{ marginBottom: '12px' }}>
          <summary style={{
            cursor: 'pointer',
            padding: '8px',
            backgroundColor: colors.surfaceAlt,
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '13px',
            fontWeight: 'bold',
            color: colors.text,
          }}>
            View Raw JSON
          </summary>
          <pre style={{
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            fontFamily: 'monospace',
            backgroundColor: colors.surfaceAlt,
            padding: '12px',
            borderRadius: '4px',
            marginTop: '8px',
            fontSize: '13px',
            overflow: 'auto',
          }}>
            {JSON.stringify(parsed, null, 2)}
          </pre>
        </details>

        {/* Show tree view */}
        <div style={{
          border: `1px solid ${colors.border}`,
          borderRadius: '4px',
          padding: '12px',
          backgroundColor: colors.background,
        }}>
          {renderNode(parsed)}
        </div>
      </div>
    );
  } catch (e) {
    // If parsing fails, return raw content
    return <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>{content}</pre>;
  }
};
