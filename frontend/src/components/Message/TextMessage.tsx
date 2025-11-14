import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface TextMessageProps {
  content: string;
}

export const TextMessage: React.FC<TextMessageProps> = ({ content }) => {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => <h1 style={{ marginTop: '16px', marginBottom: '12px', fontSize: '28px', fontWeight: 'bold' }}>{children}</h1>,
        h2: ({ children }) => <h2 style={{ marginTop: '14px', marginBottom: '10px', fontSize: '24px', fontWeight: 'bold' }}>{children}</h2>,
        h3: ({ children }) => <h3 style={{ marginTop: '12px', marginBottom: '8px', fontSize: '20px', fontWeight: 'bold' }}>{children}</h3>,
        p: ({ children }) => <p style={{ marginBottom: '12px' }}>{children}</p>,
        ul: ({ children }) => <ul style={{ marginLeft: '20px', marginBottom: '12px', paddingLeft: '20px' }}>{children}</ul>,
        ol: ({ children }) => <ol style={{ marginLeft: '20px', marginBottom: '12px', paddingLeft: '20px' }}>{children}</ol>,
        li: ({ children }) => <li style={{ marginBottom: '6px' }}>{children}</li>,
        code: ({ children }) => <code style={{ backgroundColor: 'rgba(0,0,0,0.2)', padding: '2px 6px', borderRadius: '3px', fontFamily: 'monospace', fontSize: '14px' }}>{children}</code>,
        pre: ({ children }) => <pre style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '12px', borderRadius: '6px', overflow: 'auto', marginBottom: '12px', fontFamily: 'monospace' }}>{children}</pre>,
        blockquote: ({ children }) => <blockquote style={{ borderLeft: '4px solid', paddingLeft: '12px', marginLeft: '0', marginBottom: '12px', opacity: 0.8 }}>{children}</blockquote>,
        table: ({ children }) => <table style={{ borderCollapse: 'collapse', marginBottom: '12px', border: '1px solid rgba(255,255,255,0.2)', width: '100%' }}>{children}</table>,
        thead: ({ children }) => <thead style={{ backgroundColor: 'rgba(0,0,0,0.2)', borderBottom: '2px solid rgba(255,255,255,0.3)' }}>{children}</thead>,
        tbody: ({ children }) => <tbody>{children}</tbody>,
        tr: ({ children }) => <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.1)' }}>{children}</tr>,
        th: ({ children }) => <th style={{ padding: '10px', textAlign: 'left', fontWeight: 'bold' }}>{children}</th>,
        td: ({ children }) => <td style={{ padding: '10px' }}>{children}</td>,
      }}
    >
      {content}
    </ReactMarkdown>
  );
};
