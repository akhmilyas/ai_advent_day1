import React from 'react';
import { getTheme } from '../../themes';

interface XmlMessageProps {
  content: string;
  colors: ReturnType<typeof getTheme>;
}

export const XmlMessage: React.FC<XmlMessageProps> = ({ content, colors }) => {
  try {
    const parser = new DOMParser();
    const xmlDoc = parser.parseFromString(content.trim(), 'text/xml');

    // Check for parsing errors
    const parserError = xmlDoc.querySelector('parsererror');
    if (parserError) {
      throw new Error('XML parsing failed');
    }

    // Recursive function to render XML nodes
    const renderNode = (node: Element | ChildNode, level: number = 0): React.ReactNode => {
      if (node.nodeType === Node.TEXT_NODE) {
        const text = node.textContent?.trim();
        if (!text) return null;
        return (
          <div style={{
            marginLeft: `${level * 20}px`,
            padding: '4px 8px',
            fontFamily: 'monospace',
            fontSize: '14px',
            color: colors.text,
          }}>
            {text}
          </div>
        );
      }

      if (node.nodeType === Node.ELEMENT_NODE) {
        const element = node as Element;
        const tagName = element.tagName;
        const attributes = Array.from(element.attributes);
        const children = Array.from(element.childNodes);
        const hasTextContent = children.length === 1 && children[0].nodeType === Node.TEXT_NODE;

        return (
          <div key={`${tagName}-${level}`} style={{ marginLeft: `${level * 20}px` }}>
            <div style={{
              padding: '6px 8px',
              backgroundColor: level % 2 === 0 ? colors.surfaceAlt : 'transparent',
              borderLeft: `3px solid ${colors.buttonPrimary}`,
              borderRadius: '4px',
              marginBottom: '4px',
            }}>
              <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>
                &lt;{tagName}
              </span>
              {attributes.length > 0 && (
                <span style={{ color: colors.textSecondary, fontFamily: 'monospace' }}>
                  {attributes.map(attr => ` ${attr.name}="${attr.value}"`).join('')}
                </span>
              )}
              <span style={{ color: colors.buttonPrimary, fontWeight: 'bold', fontFamily: 'monospace' }}>
                &gt;
              </span>
              {hasTextContent && (
                <span style={{ color: colors.text, fontFamily: 'monospace', marginLeft: '8px' }}>
                  {children[0].textContent}
                </span>
              )}
            </div>
            {!hasTextContent && children.length > 0 && (
              <div>
                {children.map((child, idx) => (
                  <React.Fragment key={idx}>
                    {renderNode(child, level + 1)}
                  </React.Fragment>
                ))}
              </div>
            )}
          </div>
        );
      }

      return null;
    };

    return (
      <div>
        {/* Show original XML */}
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
            View Raw XML
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
            {content}
          </pre>
        </details>

        {/* Show tree view */}
        <div style={{
          border: `1px solid ${colors.border}`,
          borderRadius: '4px',
          padding: '12px',
          backgroundColor: colors.background,
        }}>
          {renderNode(xmlDoc.documentElement)}
        </div>
      </div>
    );
  } catch (e) {
    // If parsing fails, return raw content
    return <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>{content}</pre>;
  }
};
