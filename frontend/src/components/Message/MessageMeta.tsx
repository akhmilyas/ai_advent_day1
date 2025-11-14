import React from 'react';
import { getTheme } from '../../themes';

interface MessageMetaProps {
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  totalCost?: number;
  latency?: number;
  generationTime?: number;
  colors: ReturnType<typeof getTheme>;
}

export const MessageMeta: React.FC<MessageMetaProps> = ({
  promptTokens,
  completionTokens,
  totalTokens,
  totalCost,
  latency,
  generationTime,
  colors,
}) => {
  // Only show if we have at least one metric
  if (totalTokens === undefined && totalCost === undefined && latency === undefined && generationTime === undefined) {
    return null;
  }

  return (
    <div style={{ fontSize: '10px', marginTop: '8px', opacity: 0.5, fontFamily: 'monospace', borderTop: `1px solid ${colors.border}`, paddingTop: '8px' }}>
      {totalTokens !== undefined && (
        <>
          Tokens: {totalTokens}
          {promptTokens !== undefined && completionTokens !== undefined && (
            <> (prompt: {promptTokens}, completion: {completionTokens})</>
          )}
        </>
      )}
      {totalCost !== undefined && (
        <>
          {totalTokens !== undefined && ' | '}
          Cost: ${totalCost.toFixed(6)}
        </>
      )}
      {latency !== undefined && (
        <>
          {(totalTokens !== undefined || totalCost !== undefined) && ' | '}
          Latency: {latency}ms
        </>
      )}
      {generationTime !== undefined && (
        <>
          {(totalTokens !== undefined || totalCost !== undefined || latency !== undefined) && ' | '}
          Gen Time: {generationTime}ms
        </>
      )}
    </div>
  );
};
