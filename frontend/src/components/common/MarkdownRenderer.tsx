import React, { useMemo } from 'react';
import katex from 'katex';

function escapeHtml(value?: string | null): string {
  if (!value) return '';
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderMath(formula?: string | null, displayMode = false): string {
  if (!formula || !formula.trim()) return '';
  try {
    return katex.renderToString(formula.trim(), {
      displayMode,
      throwOnError: false,
    });
  } catch {
    return escapeHtml(formula);
  }
}

function highlightText(text?: string | null, query?: string | null): string {
  if (!text) return '';
  if (!query || !query.trim()) return text;
  const q = query.trim().replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(`(${q})`, 'gi');
  return text.replace(regex, '<mark class="bg-amber-300/80 dark:bg-amber-500/50 text-slate-900 dark:text-slate-100 rounded px-0.5">$1</mark>');
}

function renderInlineMarkdown(value?: string | null, searchQuery?: string): string {
  if (!value) return '';
  const maths: string[] = [];
  let textWithoutMath = value.replace(/\$\$([\s\S]+?)\$\$/g, (_, formula) => {
    const idx = maths.length;
    maths.push(renderMath(formula, true));
    return `@@@MDMATH${idx}@@@`;
  });
  textWithoutMath = textWithoutMath.replace(/\\\[([\s\S]+?)\\\]/g, (_, formula) => {
    const idx = maths.length;
    maths.push(renderMath(formula, true));
    return `@@@MDMATH${idx}@@@`;
  });
  textWithoutMath = textWithoutMath.replace(/\$([^\n$]+?)\$/g, (_, formula) => {
    const idx = maths.length;
    maths.push(renderMath(formula, false));
    return `@@@MDMATH${idx}@@@`;
  });
  textWithoutMath = textWithoutMath.replace(/\\\(([\s\S]+?)\\\)/g, (_, formula) => {
    const idx = maths.length;
    maths.push(renderMath(formula, false));
    return `@@@MDMATH${idx}@@@`;
  });

  const codes: string[] = [];
  const textWithoutCodes = textWithoutMath.replace(/`([^`]+)`/g, (_, code) => {
    const idx = codes.length;
    codes.push(code);
    return `@@@MDCODE${idx}@@@`;
  });

  const links: { label: string; url: string }[] = [];
  const textWithPlaceholders = textWithoutCodes.replace(/\[([^\]]+)\]\(((?:https?:\/\/|\/)[^\s)]+)\)/g, (fullMatch, label, url) => {
    if (url.startsWith('//') || url.startsWith('/\\')) {
      return fullMatch;
    }
    const idx = links.length;
    links.push({ label, url });
    return `@@@MDLINK${idx}@@@`;
  });

  let html = escapeHtml(textWithPlaceholders)
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/__(.+?)__/g, '<strong>$1</strong>')
    .replace(/~~(.+?)~~/g, '<del class="opacity-75">$1</del>')
    .replace(/(?<!\*)\*(?!\*)(.+?)(?<!\*)\*(?!\*)/g, '<em>$1</em>')
    .replace(/(?<!_)_(?!_)(.+?)(?<!_)_(?!_)/g, '<em>$1</em>');

  html = html.replace(
    /(^|[\s(])(https?:\/\/[^\s)<>]+?)([.,;:!?]?(?:[\s)]|$))/g,
    '$1<a href="$2" target="_blank" rel="noopener noreferrer" class="text-tlu-blue dark:text-blue-400 underline hover:text-tlu-navy transition-colors font-medium break-all">$2</a>$3',
  );

  for (let i = 0; i < codes.length; i++) {
    const safeCode = escapeHtml(codes[i]);
    const codeTag = `<code class="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-800 font-mono text-[13px] text-tlu-navy dark:text-blue-300 font-semibold">${safeCode}</code>`;
    html = html.split(`@@@MDCODE${i}@@@`).join(codeTag);
  }

  for (let i = 0; i < links.length; i++) {
    const { label, url } = links[i];
    if (url.startsWith('//') || url.startsWith('/\\')) {
      continue;
    }
    let safeLabel = escapeHtml(label);
    safeLabel = safeLabel
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/__(.+?)__/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      .replace(/_(.+?)_/g, '<em>$1</em>');
    const safeUrl = encodeURI(url);
    const linkTag = `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer" class="text-tlu-blue dark:text-blue-400 underline hover:text-tlu-navy transition-colors font-medium break-all">${safeLabel}</a>`;
    html = html.split(`@@@MDLINK${i}@@@`).join(linkTag);
  }

  for (let i = 0; i < maths.length; i++) {
    html = html.split(`@@@MDMATH${i}@@@`).join(maths[i]);
  }

  if (searchQuery) {
    html = highlightText(html, searchQuery);
  }

  return html;
}

function parseMarkdownTableCells(line?: string | null): string[] {
  let trimmed = (line || '').trim();
  if (trimmed.startsWith('|')) trimmed = trimmed.slice(1);
  if (trimmed.endsWith('|')) trimmed = trimmed.slice(0, -1);
  return trimmed.split('|').map((c) => (c || '').trim());
}

function parseMarkdownTable(lines?: string[] | null): { headers: string[]; rows: string[][] } | null {
  const tableLines = (lines || []).map((l) => (l || '').trim()).filter((l) => l.length > 0);
  if (tableLines.length < 2) return null;
  const sepIdx = tableLines.findIndex((l) => /^\s*\|?\s*[-:]+[-| :]*\|?\s*$/.test(l));
  if (sepIdx > 0) {
    const headers = parseMarkdownTableCells(tableLines[0]);
    const rows = tableLines.slice(sepIdx + 1).map(parseMarkdownTableCells);
    return { headers, rows };
  }
  const headers = parseMarkdownTableCells(tableLines[0]);
  const rows = tableLines.slice(1).map(parseMarkdownTableCells);
  return { headers, rows };
}

type MarkdownBlock =
  | { type: 'code'; language: string; code: string }
  | { type: 'heading'; level: number; text: string }
  | { type: 'blockquote'; text: string }
  | { type: 'table'; headers: string[]; rows: string[][] }
  | { type: 'ul'; items: string[] }
  | { type: 'ol'; items: string[] }
  | { type: 'hr' }
  | { type: 'p'; lines: string[] }
  | { type: 'math'; formula: string };

function parseMarkdownBlocks(content?: string | null): MarkdownBlock[] {
  if (!content || typeof content !== 'string') return [];
  const blocks: MarkdownBlock[] = [];
  const lines = content.replace(/\r\n/g, '\n').split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];
    if (line === undefined || line === null) {
      i++;
      continue;
    }
    const trimmed = line.trim();

    if (!trimmed) {
      i++;
      continue;
    }

    if (trimmed.startsWith('$$')) {
      if (trimmed.endsWith('$$') && trimmed.length > 2) {
        blocks.push({
          type: 'math',
          formula: trimmed.slice(2, -2).trim(),
        });
        i++;
        continue;
      }
      const mathLines: string[] = [trimmed.slice(2)];
      i++;
      while (i < lines.length && !lines[i]?.trim().endsWith('$$')) {
        mathLines.push(lines[i] ?? '');
        i++;
      }
      if (i < lines.length) {
        const last = (lines[i] ?? '').trim();
        mathLines.push(last.slice(0, -2));
        i++;
      }
      blocks.push({
        type: 'math',
        formula: mathLines.join('\n').trim(),
      });
      continue;
    }

    if (trimmed.startsWith('```')) {
      const language = trimmed.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i]?.trim().startsWith('```')) {
        codeLines.push(lines[i] ?? '');
        i++;
      }
      if (i < lines.length && lines[i]?.trim().startsWith('```')) {
        i++;
      }
      blocks.push({
        type: 'code',
        language,
        code: codeLines.join('\n'),
      });
      continue;
    }

    const hMatch = line.match(/^(#{1,6})\s+(.+)$/);
    if (hMatch) {
      blocks.push({
        type: 'heading',
        level: hMatch[1].length,
        text: (hMatch[2] || '').trim(),
      });
      i++;
      continue;
    }

    if (/^(?:---|\*\*\*|___)\s*$/.test(trimmed)) {
      blocks.push({ type: 'hr' });
      i++;
      continue;
    }

    if (trimmed.startsWith('>')) {
      const bqLines: string[] = [];
      while (i < lines.length && lines[i]?.trim().startsWith('>')) {
        bqLines.push((lines[i] ?? '').trim().replace(/^>\s?/, ''));
        i++;
      }
      blocks.push({
        type: 'blockquote',
        text: bqLines.join(' '),
      });
      continue;
    }

    const nextLineTrimmed = i + 1 < lines.length ? (lines[i + 1] ?? '').trim() : '';
    if (trimmed.startsWith('|') || (trimmed.includes('|') && /^\s*\|?\s*[-:]+[-| :]*\|?\s*$/.test(nextLineTrimmed))) {
      const tableLines: string[] = [];
      while (i < lines.length && ((lines[i] ?? '').trim().startsWith('|') || (lines[i] ?? '').includes('|'))) {
        tableLines.push(lines[i] ?? '');
        i++;
      }
      const tableData = parseMarkdownTable(tableLines);
      if (tableData && tableData.headers.length > 0) {
        blocks.push({
          type: 'table',
          headers: tableData.headers,
          rows: tableData.rows,
        });
      } else if (tableLines.length > 0) {
        blocks.push({
          type: 'p',
          lines: tableLines.map((l) => (l ?? '').trim()).filter(Boolean),
        });
      }
      continue;
    }

    if (/^\s*[-*+]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length) {
        const curLine = lines[i] ?? '';
        const curTrimmed = curLine.trim();
        const ulMatch = curLine.match(/^\s*[-*+]\s+(.+)$/);
        if (ulMatch) {
          items.push(ulMatch[1]);
          i++;
        } else if (items.length > 0 && /^\s{2,}\S/.test(curLine) && !curTrimmed.startsWith('>') && !curTrimmed.startsWith('```')) {
          items[items.length - 1] += ' ' + curTrimmed;
          i++;
        } else {
          break;
        }
      }
      blocks.push({
        type: 'ul',
        items,
      });
      continue;
    }

    if (/^\s*\d+[.)]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length) {
        const curLine = lines[i] ?? '';
        const curTrimmed = curLine.trim();
        const olMatch = curLine.match(/^\s*\d+[.)]\s+(.+)$/);
        if (olMatch) {
          items.push(olMatch[1]);
          i++;
        } else if (items.length > 0 && /^\s{2,}\S/.test(curLine) && !curTrimmed.startsWith('>') && !curTrimmed.startsWith('```')) {
          items[items.length - 1] += ' ' + curTrimmed;
          i++;
        } else {
          break;
        }
      }
      blocks.push({
        type: 'ol',
        items,
      });
      continue;
    }

    const prevI = i;
    const pLines: string[] = [];
    while (i < lines.length) {
      const curLine = lines[i] ?? '';
      const curTrimmed = curLine.trim();
      if (!curTrimmed) break;
      if (curTrimmed.startsWith('```') || /^(?:---|\*\*\*|___)\s*$/.test(curTrimmed)) break;
      if (/^#{1,6}\s+/.test(curTrimmed)) break;
      if (curTrimmed.startsWith('>')) break;
      if (/^\s*[-*+]\s+/.test(curLine)) break;
      if (/^\s*\d+[.)]\s+/.test(curLine)) break;
      if (curTrimmed.startsWith('|') || curTrimmed.startsWith('$$') || curTrimmed.startsWith('\\[')) break;

      pLines.push(curTrimmed);
      i++;
    }

    if (pLines.length > 0) {
      blocks.push({
        type: 'p',
        lines: pLines,
      });
    }

    if (i <= prevI) {
      if (i < lines.length && lines[i] !== undefined) {
        const fallback = (lines[i] ?? '').trim();
        if (fallback) {
          blocks.push({
            type: 'p',
            lines: [fallback],
          });
        }
      }
      i = Math.max(i + 1, prevI + 1);
    }
  }

  return blocks;
}

interface MarkdownRendererProps {
  content?: string | null;
  className?: string;
  searchQuery?: string;
}

export const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({
  content,
  className = '',
  searchQuery,
}) => {
  const blocks = useMemo(() => parseMarkdownBlocks(content), [content]);

  return (
    <div className={`space-y-3 leading-relaxed text-sm wrap-break-word ${className}`}>
      {blocks.map((b, idx) => {
        if (b.type === 'code') {
          return (
            <div
              key={idx}
              className="my-3 rounded-xl overflow-hidden bg-slate-900 border border-slate-800 text-slate-100 shadow-xs font-mono text-xs"
            >
              {b.language && (
                <div className="px-4 py-1.5 bg-slate-800/80 border-b border-slate-700/60 text-[11px] text-slate-400 font-semibold uppercase tracking-wider flex items-center justify-between">
                  <span>{b.language}</span>
                </div>
              )}
              <pre className="p-4 overflow-x-auto">
                <code>{b.code}</code>
              </pre>
            </div>
          );
        }

        if (b.type === 'heading') {
          if (b.level === 1) {
            return (
              <h1
                key={idx}
                className="text-lg sm:text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-4 mb-2 pb-1.5 border-b border-slate-200 dark:border-slate-800"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text, searchQuery) }}
              />
            );
          }
          if (b.level === 2) {
            return (
              <h2
                key={idx}
                className="text-base font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-3 mb-1.5"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text, searchQuery) }}
              />
            );
          }
          if (b.level === 3) {
            return (
              <h3
                key={idx}
                className="text-sm font-bold text-slate-900 dark:text-slate-100 mt-2 mb-1"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text, searchQuery) }}
              />
            );
          }
          return (
            <h4
              key={idx}
              className="text-xs font-bold uppercase tracking-wider text-slate-800 dark:text-slate-200 mt-2 mb-1"
              dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text, searchQuery) }}
            />
          );
        }

        if (b.type === 'hr') {
          return <hr key={idx} className="my-3 border-slate-200 dark:border-slate-800" />;
        }

        if (b.type === 'math') {
          return (
            <div
              key={idx}
              className="my-3 py-1.5 overflow-x-auto text-center"
              dangerouslySetInnerHTML={{ __html: renderMath(b.formula, true) }}
            />
          );
        }

        if (b.type === 'blockquote') {
          return (
            <blockquote
              key={idx}
              className="my-3 pl-3.5 border-l-3 border-tlu-blue dark:border-tlu-cyan py-1 text-slate-600 dark:text-slate-300 bg-blue-50/40 dark:bg-blue-950/20 rounded-r-lg italic text-xs sm:text-sm"
              dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text, searchQuery) }}
            />
          );
        }

        if (b.type === 'table') {
          return (
            <div
              key={idx}
              className="my-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800 shadow-2xs"
            >
              <table className="min-w-full text-xs text-left divide-y divide-slate-200 dark:divide-slate-800">
                <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-700 dark:text-slate-200 font-semibold">
                  <tr>
                    {(b.headers || []).map((h, hIdx) => (
                      <th
                        key={hIdx}
                        className="px-3.5 py-2.5 whitespace-nowrap"
                        dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(h, searchQuery) }}
                      />
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800 bg-white dark:bg-slate-900/60">
                  {(b.rows || []).map((row, rIdx) => (
                    <tr
                      key={rIdx}
                      className={rIdx % 2 === 1 ? 'bg-slate-50/50 dark:bg-slate-800/30' : ''}
                    >
                      {(row || []).map((cell, cIdx) => (
                        <td
                          key={cIdx}
                          className="px-3.5 py-2 whitespace-normal leading-relaxed text-slate-600 dark:text-slate-300"
                          dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(cell, searchQuery) }}
                        />
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        }

        if (b.type === 'ul') {
          return (
            <ul key={idx} className="my-2 space-y-1 pl-5 list-disc text-slate-700 dark:text-slate-300 text-xs sm:text-sm">
              {(b.items || []).map((item, iIdx) => (
                <li
                  key={iIdx}
                  dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(item, searchQuery) }}
                />
              ))}
            </ul>
          );
        }

        if (b.type === 'ol') {
          return (
            <ol key={idx} className="my-2 space-y-1 pl-5 list-decimal text-slate-700 dark:text-slate-300 text-xs sm:text-sm">
              {(b.items || []).map((item, iIdx) => (
                <li
                  key={iIdx}
                  dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(item, searchQuery) }}
                />
              ))}
            </ol>
          );
        }

        return (
          <p
            key={idx}
            className="text-slate-700 dark:text-slate-300 leading-relaxed text-xs sm:text-sm"
            dangerouslySetInnerHTML={{
              __html: renderInlineMarkdown((b.lines || []).join('\n'), searchQuery).replace(/\n/g, '<br />'),
            }}
          />
        );
      })}
    </div>
  );
};
