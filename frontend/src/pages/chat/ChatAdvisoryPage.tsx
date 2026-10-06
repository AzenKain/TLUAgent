import React, { useRef, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Sparkles,
  BookOpen,
  Plus,
  X,
  Copy,
  Check,
  ThumbsUp,
  ThumbsDown,
  Square,
  ChevronDown,
  FileText,
  Award,
  Calculator,
  GraduationCap,
  RefreshCw,
  Paperclip,
  PanelLeft,
  PanelLeftClose,
  Trash2,
  LogIn,
  LogOut,
  LayoutDashboard,
  User as UserIcon,
  ArrowUp,
  Edit3,
  Brain,
  CheckCircle2,
  Send,
} from 'lucide-react';
import { chatService, type ChatMessage } from '@/services/chatService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { useChatStore } from '@/stores/chatStore';
import { useAuth } from '@/hooks/useAuth';
import { usePermissions } from '@/hooks/usePermissions';
import { useShallow } from 'zustand/react/shallow';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { Modal } from '@/components/common/Modal';
import { NotificationBell } from '@/components/notifications/NotificationBell';
import { inquiryService } from '@/services/inquiryService';
import katex from 'katex';

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderMath(formula: string, displayMode: boolean): string {
  try {
    return katex.renderToString(formula.trim(), {
      displayMode,
      throwOnError: false,
    });
  } catch {
    return escapeHtml(formula);
  }
}

function renderInlineMarkdown(value: string): string {
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
  const textWithPlaceholders = textWithoutCodes.replace(/\[([^\]]+)\]\(((?:https?:\/\/|\/)[^\s)]+)\)/g, (_, label, url) => {
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
    '$1<a href="$2" target="_blank" rel="noopener noreferrer" class="text-indigo-600 dark:text-indigo-400 underline hover:text-indigo-500 transition-colors font-medium break-all">$2</a>$3',
  );

  for (let i = 0; i < codes.length; i++) {
    const safeCode = escapeHtml(codes[i]);
    const codeTag = `<code class="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-800 font-mono text-[13px] text-indigo-600 dark:text-indigo-400">${safeCode}</code>`;
    html = html.split(`@@@MDCODE${i}@@@`).join(codeTag);
  }

  for (let i = 0; i < links.length; i++) {
    const { label, url } = links[i];
    let safeLabel = escapeHtml(label);
    safeLabel = safeLabel
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/__(.+?)__/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      .replace(/_(.+?)_/g, '<em>$1</em>');
    const safeUrl = encodeURI(url);
    const linkTag = `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer" class="text-indigo-600 dark:text-indigo-400 underline hover:text-indigo-500 transition-colors font-medium break-all">${safeLabel}</a>`;
    html = html.split(`@@@MDLINK${i}@@@`).join(linkTag);
  }

  for (let i = 0; i < maths.length; i++) {
    html = html.split(`@@@MDMATH${i}@@@`).join(maths[i]);
  }

  return html;
}

function normalizeMarkdownTables(text: string): string {
  return text;
}

function parseMarkdownTableCells(line: string): string[] {
  return line
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((c) => c.trim());
}

function parseMarkdownTable(lines: string[]): { headers: string[]; rows: string[][] } | null {
  const tableLines = lines.map((l) => l.trim()).filter((l) => l.startsWith('|'));
  if (tableLines.length === 0) return null;

  const sepIdx = tableLines.findIndex((l) => /^\|(\s*:?-+:?\s*\|?)+$/.test(l));
  if (sepIdx > 0) {
    const headers = parseMarkdownTableCells(tableLines[0]);
    const rows = tableLines.slice(sepIdx + 1).map(parseMarkdownTableCells);
    return { headers, rows };
  }

  const headers = parseMarkdownTableCells(tableLines[0]);
  const rows = tableLines.slice(1).map(parseMarkdownTableCells);
  return { headers, rows };
}

function extractStudentCodeFromText(text: string): string {
  const match = text.match(/\b([A-Za-z]\d{5})\b/);
  return match ? match[1].toUpperCase() : '';
}

function extractStudentClassFromText(text: string): string {
  const match = text.match(/\b(?:k|kho\u00e1)\s*(\d{2,3})\b/i);
  if (match) return `K${match[1]}`;
  const classMatch = text.match(/\b([A-Za-z0-9_-]{2,15}-K\d{2})\b/i);
  if (classMatch) return classMatch[1].toUpperCase();
  return '';
}

function buildFormalStudentQuestion(
  rawQuery: string,
  name: string,
  code: string,
  className: string,
  translate: (key: string, opts?: Record<string, string>) => string,
): string {
  const prefixRegex = new RegExp(
    '^(?:cho\\s+t\\u00f4i\\s+h\\u1ecfi\\s*(?:v\\u1ec1|v\\u1edbi)?|cho\\s+em\\s+h\\u1ecfi\\s*(?:v\\u1ec1|v\\u1edbi)?|cho\\s+m\\u00ecnh\\s+h\\u1ecfi\\s*(?:v\\u1ec1|v\\u1edbi)?|t\\u00f4i\\s+mu\\u1ed1n\\s+h\\u1ecfi\\s*(?:v\\u1ec1)?|em\\s+mu\\u1ed1n\\s+h\\u1ecfi\\s*(?:v\\u1ec1)?|h\\u1ecfi\\s+v\\u1ec1)\\s*',
    'i',
  );
  const suffixRegex = new RegExp('[\\s,]+(?:\\u1ea1|nh\\u00e9|v\\u1edbi|nha|v\\u1eady|kh\\u00f4ng|ko)\\s*$', 'i');
  const clean = rawQuery.replace(prefixRegex, '').replace(suffixRegex, '').trim();

  const topic = clean ? clean.charAt(0).toUpperCase() + clean.slice(1) : rawQuery.trim();
  const studentName = name.trim() || translate('inquiries.student_placeholder_name');
  const studentCode = code.trim() || translate('inquiries.student_placeholder_code');
  const studentClass = className.trim() || translate('inquiries.student_placeholder_class');

  return [
    translate('inquiries.formal_template_salutation'),
    '',
    translate('inquiries.formal_template_intro', {
      name: studentName,
      code: studentCode,
      class: studentClass,
    }),
    '',
    translate('inquiries.formal_template_body_prefix'),
    `- ${topic}.`,
    '',
    translate('inquiries.formal_template_closing'),
  ].join('\n');
}

function buildFormalStudentContext(
  rawContext: string,
  translate: (key: string) => string,
): string {
  if (!rawContext.trim()) return '';
  const greetingRegex = new RegExp('^Ch\\u00e0o b\\u1ea1n [^,\\n]+,\\s*', 'i');
  const thankRegex = new RegExp('^C\\u1ea3m \\u01a1n b\\u1ea1n \\u0111\\u00e3 li\\u00ean h\\u1ec7[^.\\n]+\\.\\s*', 'i');
  const clean = rawContext
    .replace(greetingRegex, '')
    .replace(thankRegex, '')
    .trim()
    .slice(0, 500);

  return translate('inquiries.formal_context_prefix') + clean;
}

function shouldSuggestEscalation(m: ChatMessage): boolean {
  if (m.sender !== 'assistant') return false;
  return Boolean(m.canEscalate);
}

type MarkdownBlock =
  | { type: 'code'; language: string; code: string }
  | { type: 'math'; formula: string }
  | { type: 'heading'; level: number; text: string }
  | { type: 'hr' }
  | { type: 'table'; headers: string[]; rows: string[][] }
  | { type: 'quote'; lines: string[] }
  | { type: 'ul'; items: string[] }
  | { type: 'ol'; items: string[] }
  | { type: 'p'; lines: string[] };

function parseMarkdownBlocks(content: string): MarkdownBlock[] {
  const blocks: MarkdownBlock[] = [];
  const lines = content.replace(/\r\n/g, '\n').split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];
    const trimmed = line.trim();

    if (!trimmed) {
      i++;
      continue;
    }

    if (trimmed.startsWith('```')) {
      const language = trimmed.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trim().startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      if (i < lines.length) {
        i++;
      }
      blocks.push({
        type: 'code',
        language,
        code: codeLines.join('\n'),
      });
      continue;
    }

    if (/^(?:---|\*\*\*|___)\s*$/.test(trimmed)) {
      blocks.push({ type: 'hr' });
      i++;
      continue;
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      blocks.push({
        type: 'heading',
        level: headingMatch[1].length,
        text: headingMatch[2].trim(),
      });
      i++;
      continue;
    }

    if (trimmed.startsWith('$$')) {
      if (trimmed.endsWith('$$') && trimmed.length > 2) {
        const formula = trimmed.slice(2, -2).trim();
        blocks.push({ type: 'math', formula });
        i++;
        continue;
      }
      const mathLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trim().startsWith('$$')) {
        mathLines.push(lines[i]);
        i++;
      }
      if (i < lines.length) {
        i++;
      }
      blocks.push({
        type: 'math',
        formula: mathLines.join('\n').trim(),
      });
      continue;
    }

    if (trimmed.startsWith('\\[') && trimmed.endsWith('\\]') && trimmed.length > 4) {
      const formula = trimmed.slice(2, -2).trim();
      blocks.push({ type: 'math', formula });
      i++;
      continue;
    }

    const prevI = i;

    if (trimmed.startsWith('|')) {
      const tableLines: string[] = [];
      let j = i;
      while (j < lines.length && lines[j].trim().startsWith('|')) {
        tableLines.push(lines[j].trim());
        j++;
      }
      const tableData = parseMarkdownTable(tableLines);
      if (tableData && tableData.headers.length > 0) {
        blocks.push({
          type: 'table',
          headers: tableData.headers,
          rows: tableData.rows,
        });
      } else {
        blocks.push({
          type: 'p',
          lines: tableLines,
        });
      }
      i = j;
      continue;
    }

    if (trimmed.startsWith('>')) {
      const quoteLines: string[] = [];
      while (i < lines.length && lines[i].trim().startsWith('>')) {
        quoteLines.push(lines[i].trim().replace(/^>\s*/, ''));
        i++;
      }
      blocks.push({
        type: 'quote',
        lines: quoteLines,
      });
      continue;
    }

    if (/^\s*[-*+]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length) {
        const curLine = lines[i];
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
        const curLine = lines[i];
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

    const pLines: string[] = [];
    while (i < lines.length) {
      const curLine = lines[i];
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
      blocks.push({
        type: 'p',
        lines: [lines[i].trim() || lines[i]],
      });
      i = prevI + 1;
    }
  }

  return blocks;
}

function FormattedMessageContent({ content, isStreaming }: { content: string; isStreaming?: boolean }) {
  if (!content) {
    if (isStreaming) {
      return <span className="inline-block w-1.5 h-4 ml-0.5 rounded-xs bg-indigo-500 animate-pulse align-middle" />;
    }
    return null;
  }

  const normalized = normalizeMarkdownTables(content.replace(/\[CAN_ESCALATE\]/g, '').trimEnd());
  const blocks = parseMarkdownBlocks(normalized);

  return (
    <div className="space-y-3 leading-relaxed text-sm break-words">
      {blocks.map((b, idx) => {
        if (b.type === 'code') {
          return (
            <div
              key={idx}
              className="my-3 rounded-2xl overflow-hidden bg-slate-900 border border-slate-800 text-slate-100 shadow-md font-mono text-xs"
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
                className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-4 mb-2"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text) }}
              />
            );
          }
          if (b.level === 2) {
            return (
              <h2
                key={idx}
                className="text-base font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-3 mb-1.5"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text) }}
              />
            );
          }
          if (b.level === 3) {
            return (
              <h3
                key={idx}
                className="text-sm font-bold text-slate-900 dark:text-slate-100 mt-2 mb-1"
                dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text) }}
              />
            );
          }
          return (
            <h4
              key={idx}
              className="text-xs font-bold uppercase tracking-wider text-slate-800 dark:text-slate-200 mt-2 mb-1"
              dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(b.text) }}
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

        if (b.type === 'table') {
          return (
            <div key={idx} className="my-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs">
              <table className="w-full text-left border-collapse text-xs">
                <thead className="bg-slate-100 dark:bg-slate-800/80 text-slate-800 dark:text-slate-200 border-b border-slate-200 dark:border-slate-700">
                  <tr>
                    {b.headers.map((h, hIdx) => (
                      <th
                        key={hIdx}
                        className="px-3.5 py-2.5 font-semibold tracking-wide"
                        dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(h) }}
                      />
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800 bg-white dark:bg-slate-900/60">
                  {b.rows.map((row, rIdx) => (
                    <tr key={rIdx} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 transition-colors">
                      {row.map((cell, cIdx) => (
                        <td
                          key={cIdx}
                          className="px-3.5 py-2.5 text-slate-700 dark:text-slate-300"
                          dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(cell) }}
                        />
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        }

        if (b.type === 'quote') {
          return (
            <blockquote
              key={idx}
              className="border-l-3 border-indigo-500 pl-3 py-1.5 my-2 text-slate-600 dark:text-slate-400 bg-slate-50/60 dark:bg-slate-800/30 rounded-r-lg"
              dangerouslySetInnerHTML={{
                __html: renderInlineMarkdown(b.lines.join('\n')).replace(/\n/g, '<br />'),
              }}
            />
          );
        }

        if (b.type === 'ul') {
          return (
            <ul key={idx} className="list-disc list-inside space-y-1 my-1.5 pl-1 text-slate-700 dark:text-slate-300">
              {b.items.map((item, iIdx) => (
                <li key={iIdx} className="leading-normal">
                  <span dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(item) }} />
                </li>
              ))}
            </ul>
          );
        }

        if (b.type === 'ol') {
          return (
            <ol key={idx} className="list-decimal list-inside space-y-1 my-1.5 pl-1 text-slate-700 dark:text-slate-300">
              {b.items.map((item, iIdx) => (
                <li key={iIdx} className="leading-normal">
                  <span dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(item) }} />
                </li>
              ))}
            </ol>
          );
        }

        return (
          <p
            key={idx}
            className="text-slate-700 dark:text-slate-300 leading-relaxed"
            dangerouslySetInnerHTML={{
              __html: renderInlineMarkdown(b.lines.join('\n')).replace(/\n/g, '<br />'),
            }}
          />
        );
      })}

      {isStreaming && (
        <span className="inline-block w-1.5 h-4 ml-0.5 rounded-xs bg-indigo-500 animate-pulse align-middle" />
      )}
    </div>
  );
}

function getFormattedTimeString(): string {
  return new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function getUniqueMessageId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
}

// ChatAdvisoryPage provides the full Google Gemini and ChatGPT style AI workspace.
export const ChatAdvisoryPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useAuth();
  const { canAny, isAdmin } = usePermissions();

  const canManage = canAny([
    'user.manage',
    'role.manage',
    'setting.manage',
    'llm.manage',
    'admin.access',
  ]) || isAdmin;

  const {
    sessions,
    activeSessionId,
    messages,
    input,
    isLoading,
    isSessionLoading,
    models,
    selectedModel,
    attachedImages,
    copiedId,
    feedbackMap,
    isModelDropdownOpen,
    isSidebarOpen,
    isUserMenuOpen,
    setInput,
    setLoading,
    addMessage,
    createNewChat,
    selectSession,
    deleteSession,
    renameSession,
    setModels,
    setSelectedModel,
    addAttachedImage,
    removeAttachedImage,
    clearAttachedImages,
    updateMessage,
    appendMessageChunk,
    appendMessageReasoning,
    updateMessageToolCall,
    setActiveSessionId,
    setCopiedId,
    setFeedback,
    setIsModelDropdownOpen,
    toggleSidebar,
    setIsUserMenuOpen,
    syncAuthMode,
    loadUserSessions,
  } = useChatStore(
    useShallow((s) => ({
      sessions: s.sessions,
      activeSessionId: s.activeSessionId,
      messages: s.messages,
      input: s.input,
      isLoading: s.isLoading,
      models: s.models,
      selectedModel: s.selectedModel,
      attachedImages: s.attachedImages,
      copiedId: s.copiedId,
      feedbackMap: s.feedbackMap,
      isModelDropdownOpen: s.isModelDropdownOpen,
      isSidebarOpen: s.isSidebarOpen,
      isUserMenuOpen: s.isUserMenuOpen,
      isSessionLoading: s.isSessionLoading,
      setInput: s.setInput,
      setLoading: s.setLoading,
      addMessage: s.addMessage,
      createNewChat: s.createNewChat,
      selectSession: s.selectSession,
      deleteSession: s.deleteSession,
      renameSession: s.renameSession,
      setModels: s.setModels,
      setSelectedModel: s.setSelectedModel,
      addAttachedImage: s.addAttachedImage,
      removeAttachedImage: s.removeAttachedImage,
      clearAttachedImages: s.clearAttachedImages,
      updateMessage: s.updateMessage,
      appendMessageChunk: s.appendMessageChunk,
      appendMessageReasoning: s.appendMessageReasoning,
      updateMessageToolCall: s.updateMessageToolCall,
      setActiveSessionId: s.setActiveSessionId,
      setCopiedId: s.setCopiedId,
      setFeedback: s.setFeedback,
      setIsModelDropdownOpen: s.setIsModelDropdownOpen,
      toggleSidebar: s.toggleSidebar,
      setIsUserMenuOpen: s.setIsUserMenuOpen,
      syncAuthMode: s.syncAuthMode,
      loadUserSessions: s.loadUserSessions,
    }))
  );

  const [expandedReasoning, setExpandedReasoning] = useState<Record<string, boolean>>({});
  const toggleReasoning = (id: string) => {
    setExpandedReasoning((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
  };

  const [editingSessionId, setEditingSessionId] = React.useState<string | null>(null);
  const [editTitle, setEditTitle] = React.useState('');
  const [isDrawerOpen, setIsDrawerOpen] = React.useState(false);
  const [isEscalateModalOpen, setIsEscalateModalOpen] = useState(false);
  const [escalateStudentName, setEscalateStudentName] = useState('');
  const [escalateStudentCode, setEscalateStudentCode] = useState('');
  const [escalateStudentClass, setEscalateStudentClass] = useState('');
  const [escalateQuestion, setEscalateQuestion] = useState('');
  const [escalateContext, setEscalateContext] = useState('');
  const [rawUserQuery, setRawUserQuery] = useState('');
  const [isQuestionCustomized, setIsQuestionCustomized] = useState(false);
  const [hasAutoFilled, setHasAutoFilled] = useState(false);
  const [formSubmitted, setFormSubmitted] = useState(false);
  const [isSubmittingEscalate, setIsSubmittingEscalate] = useState(false);
  const [escalateToast, setEscalateToast] = useState<string | null>(null);
  const [submittedEscalateIds, setSubmittedEscalateIds] = useState<Set<string>>(new Set());
  const [activeEscalateMsgId, setActiveEscalateMsgId] = useState<string | null>(null);

  const handleNameChange = (newName: string) => {
    setEscalateStudentName(newName);
    if (!isQuestionCustomized) {
      setEscalateQuestion(buildFormalStudentQuestion(rawUserQuery, newName, escalateStudentCode, escalateStudentClass, t));
    }
  };

  const handleCodeChange = (newCode: string) => {
    const upper = newCode.toUpperCase();
    setEscalateStudentCode(upper);
    if (!isQuestionCustomized) {
      setEscalateQuestion(buildFormalStudentQuestion(rawUserQuery, escalateStudentName, upper, escalateStudentClass, t));
    }
  };

  const handleClassChange = (newClass: string) => {
    setEscalateStudentClass(newClass);
    if (!isQuestionCustomized) {
      setEscalateQuestion(buildFormalStudentQuestion(rawUserQuery, escalateStudentName, escalateStudentCode, newClass, t));
    }
  };

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const reasoningScrollRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const modelDropdownRef = useRef<HTMLDivElement>(null);
  const userMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    syncAuthMode(isAuthenticated);
  }, [isAuthenticated, syncAuthMode]);

  useEffect(() => {
    chatService.getModels().then(setModels).catch(() => {});
  }, [setModels]);

  useEffect(() => {
    if (messages.length > 0) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
    if (reasoningScrollRef.current) {
      reasoningScrollRef.current.scrollTop = reasoningScrollRef.current.scrollHeight;
    }
  }, [messages]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (modelDropdownRef.current && !modelDropdownRef.current.contains(e.target as Node)) {
        setIsModelDropdownOpen(false);
      }
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setIsUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [setIsModelDropdownOpen, setIsUserMenuOpen]);

  useEffect(() => {
    if (!isDrawerOpen) return;
    const handleEscapeKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsDrawerOpen(false);
      }
    };
    document.addEventListener('keydown', handleEscapeKey);
    return () => document.removeEventListener('keydown', handleEscapeKey);
  }, [isDrawerOpen]);

  useEffect(() => {
    if (isAuthenticated || !activeSessionId || !activeSessionId.startsWith('guest_')) return;
    const handleLeave = () => {
      chatService.notifyGuestLeave(activeSessionId);
    };
    window.addEventListener('pagehide', handleLeave);
    return () => {
      window.removeEventListener('pagehide', handleLeave);
    };
  }, [isAuthenticated, activeSessionId]);

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 180)}px`;
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    const maxAllowed = selectedModel?.max_images || 5;
    if (attachedImages.length >= maxAllowed) {
      alert(t('chat.max_images_reached', { max: maxAllowed }));
      return;
    }

    const file = files[0];
    const reader = new FileReader();
    reader.addEventListener('load', () => {
      const base64 = reader.result as string;
      addAttachedImage(base64);
    });
    reader.readAsDataURL(file);
    e.target.value = '';
  };

  const handleStopGeneration = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setLoading(false);
  };

  const handleOpenEscalateForMessage = (targetMsg?: ChatMessage) => {
    setActiveEscalateMsgId(targetMsg?.id || null);
    setFormSubmitted(false);
    setIsQuestionCustomized(false);
    let q = '';
    let c = '';

    if (targetMsg) {
      const msgIdx = messages.findIndex((m) => m.id === targetMsg.id);
      const prevUserMsg = messages
        .slice(0, msgIdx >= 0 ? msgIdx : undefined)
        .findLast((m) => m.sender === 'user');
      q = prevUserMsg?.text || (input.trim() ? input.trim() : '');
      c = targetMsg.text;
    } else {
      const lastUserMsg = messages.findLast((m) => m.sender === 'user');
      q = input.trim() || lastUserMsg?.text || '';
    }

    const combinedText = messages.map((m) => m.text).join(' ') + ' ' + input;
    const foundName = user?.full_name || '';
    const foundCode = user?.student_code || extractStudentCodeFromText(combinedText) || '';
    const foundClass = extractStudentClassFromText(combinedText) || '';

    setRawUserQuery(q);
    setEscalateStudentName(foundName);
    setEscalateStudentCode(foundCode);
    setEscalateStudentClass(foundClass);

    setEscalateQuestion(buildFormalStudentQuestion(q, foundName, foundCode, foundClass, t));
    setEscalateContext(buildFormalStudentContext(c, t));

    const didAutoFill = Boolean(foundName || foundCode || foundClass || q);
    setHasAutoFilled(didAutoFill);
    setIsEscalateModalOpen(true);
  };

  const handleSendEscalate = async () => {
    setFormSubmitted(true);
    if (
      !escalateStudentName.trim() ||
      !escalateStudentCode.trim() ||
      !escalateStudentClass.trim() ||
      !escalateQuestion.trim()
    ) {
      return;
    }
    setIsSubmittingEscalate(true);
    try {
      await inquiryService.createInquiry({
        student_name: escalateStudentName.trim(),
        student_code: escalateStudentCode.trim(),
        student_class: escalateStudentClass.trim(),
        question: escalateQuestion.trim(),
        context: escalateContext.trim() || undefined,
        conversation_id: activeSessionId || undefined,
      });
      if (activeEscalateMsgId) {
        setSubmittedEscalateIds((prev) => new Set(prev).add(activeEscalateMsgId));
      }
      setEscalateToast(t('inquiries.escalate_success'));
      setIsEscalateModalOpen(false);
      setEscalateQuestion('');
      setEscalateContext('');
      setFormSubmitted(false);
      setActiveEscalateMsgId(null);
      setTimeout(() => setEscalateToast(null), 4000);
    } catch {
      setEscalateToast(t('common.error'));
      setTimeout(() => setEscalateToast(null), 4000);
    } finally {
      setIsSubmittingEscalate(false);
    }
  };

  const handleExecuteSend = async (queryText: string) => {
    const query = queryText.trim();
    if (!query || isLoading) return;

    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    const currentTime = getFormattedTimeString();
    const currentImages = [...attachedImages];

    const userMsg: ChatMessage = {
      id: getUniqueMessageId(),
      sender: 'user',
      text: query,
      images: currentImages.length > 0 ? currentImages : undefined,
      timestamp: currentTime,
    };

    addMessage(userMsg);
    setInput('');
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
    clearAttachedImages();
    setLoading(true);

    const botMsgId = getUniqueMessageId();
    const initialBotMsg: ChatMessage = {
      id: botMsgId,
      sender: 'assistant',
      text: '',
      timestamp: getFormattedTimeString(),
    };
    addMessage(initialBotMsg);

    const historyPayload = activeSessionId
      ? undefined
      : messages
          .filter((m) => m.id !== 'welcome')
          .slice(-10)
          .map((m) => ({
            role: m.sender,
            content: m.text.slice(0, 30000),
          }));

    try {
      await chatService.askAdvisorStream(
        query,
        selectedModel?.id,
        currentImages.length > 0 ? currentImages : undefined,
        historyPayload && historyPayload.length > 0 ? historyPayload : undefined,
        activeSessionId || undefined,
        {
          onChunk: (chunk) => {
            appendMessageChunk(botMsgId, chunk);
          },
          onReasoning: (chunk) => {
            appendMessageReasoning(botMsgId, chunk);
          },
          onToolCall: (tc) => {
            updateMessageToolCall(botMsgId, tc);
          },
          onSources: (sources) => {
            updateMessage(botMsgId, { sources });
          },
          onCanEscalate: (canEscalate) => {
            updateMessage(botMsgId, { canEscalate });
          },
          onRehydrateRequired: async () => {
            const rehydrateHistory = messages
              .filter((m) => m.id !== 'welcome' && m.id !== botMsgId)
              .slice(-20)
              .map((m) => ({
                role: m.sender,
                content: m.text.slice(0, 30000),
              }));
            try {
              await chatService.askAdvisorStream(
                query,
                selectedModel?.id,
                currentImages.length > 0 ? currentImages : undefined,
                rehydrateHistory,
                activeSessionId || undefined,
                {
                  onChunk: (chunk) => appendMessageChunk(botMsgId, chunk),
                  onReasoning: (chunk) => appendMessageReasoning(botMsgId, chunk),
                  onToolCall: (tc) => updateMessageToolCall(botMsgId, tc),
                  onSources: (sources) => updateMessage(botMsgId, { sources }),
                  onCanEscalate: (canEscalate) => updateMessage(botMsgId, { canEscalate }),
                  onDone: (ts, returnedSessionId) => {
                    if (ts) updateMessage(botMsgId, { timestamp: ts });
                    if (returnedSessionId && activeSessionId !== returnedSessionId) {
                      setActiveSessionId(returnedSessionId);
                    }
                  },
                },
                abortController.signal
              );
            } catch {}
          },
          onDone: (ts, returnedSessionId) => {
            if (ts) {
              updateMessage(botMsgId, { timestamp: ts });
            }
            if (returnedSessionId) {
              if (activeSessionId !== returnedSessionId) {
                setActiveSessionId(returnedSessionId);
              }
              if (isAuthenticated) {
                loadUserSessions();
              }
            }
          },
        },
        abortController.signal
      );
    } catch (err: unknown) {
      if ((err as Error)?.name === 'AbortError') {
        return;
      }
      updateMessage(botMsgId, {
        text: t('chat.error_reply'),
        timestamp: getFormattedTimeString(),
      });
    } finally {
      setLoading(false);
      abortControllerRef.current = null;
    }
  };

  const handleSend = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    handleExecuteSend(input);
  };

  const handleCopyMessage = (id: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => {
      setCopiedId(null);
    }, 2000);
  };

  const handleFeedback = (id: string, type: 'up' | 'down') => {
    setFeedback(id, type);
  };

  const handleRegenerate = () => {
    const userMsgs = messages.filter((m) => m.sender === 'user');
    if (userMsgs.length === 0) return;
    const lastUserMsg = userMsgs[userMsgs.length - 1];
    handleExecuteSend(lastUserMsg.text);
  };

  const suggestions = [
    {
      icon: BookOpen,
      color: 'text-blue-500 bg-blue-50 dark:bg-blue-950/50 border-blue-200 dark:border-blue-900',
      title: t('chat.suggestion_1_title'),
      desc: t('chat.suggestion_1_desc'),
    },
    {
      icon: Award,
      color: 'text-amber-500 bg-amber-50 dark:bg-amber-950/50 border-amber-200 dark:border-amber-900',
      title: t('chat.suggestion_2_title'),
      desc: t('chat.suggestion_2_desc'),
    },
    {
      icon: Calculator,
      color: 'text-emerald-500 bg-emerald-50 dark:bg-emerald-950/50 border-emerald-200 dark:border-emerald-900',
      title: t('chat.suggestion_3_title'),
      desc: t('chat.suggestion_3_desc'),
    },
    {
      icon: GraduationCap,
      color: 'text-violet-500 bg-violet-50 dark:bg-violet-950/50 border-violet-200 dark:border-violet-900',
      title: t('chat.suggestion_4_title'),
      desc: t('chat.suggestion_4_desc'),
    },
  ];

  const hasMessages = messages.length > 0 && !(messages.length === 1 && messages[0].id === 'welcome');

  return (
    <div className="flex h-dvh w-full overflow-hidden bg-white dark:bg-[#131314] text-slate-900 dark:text-slate-100 select-text">
      {/* Collapsible Chat Sidebar */}
      {isDrawerOpen && (
        <div
          className="fixed inset-0 z-40 bg-slate-900/50 md:hidden"
          onClick={() => setIsDrawerOpen(false)}
        />
      )}
      <aside
        aria-label={t('chat.recent_history')}
        className={`fixed inset-y-0 left-0 z-50 h-full w-72 max-w-[85vw] bg-[#f9fafb] dark:bg-[#18191a] border-r border-slate-200/80 dark:border-slate-800 flex flex-col shrink-0 transform transition-all duration-300 md:relative md:z-30 md:max-w-none ${
          isDrawerOpen ? 'translate-x-0 visible' : '-translate-x-full invisible md:translate-x-0 md:visible'
        } ${
          isSidebarOpen ? 'md:w-64' : 'md:w-0 md:overflow-hidden md:border-r-0'
        }`}
      >
        <div className="p-3.5 flex items-center justify-between border-b border-slate-200/60 dark:border-slate-800/80">
          <button
            onClick={() => {
              createNewChat();
              setIsDrawerOpen(false);
            }}
            className="flex-1 flex items-center gap-2 px-3.5 py-2.5 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700/80 text-xs font-semibold text-slate-800 dark:text-slate-200 shadow-xs hover:bg-slate-50 dark:hover:bg-slate-800 transition"
          >
            <Plus className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
            <span>{t('chat.new_chat')}</span>
          </button>
          <button
            onClick={() => setIsDrawerOpen(false)}
            title={t('chat.close_sidebar')}
            aria-label={t('chat.close_sidebar')}
            className="ml-2 flex h-11 w-11 items-center justify-center rounded-xl text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-800 transition md:hidden"
          >
            <X className="w-5 h-5" />
          </button>
          <button
            onClick={toggleSidebar}
            title={t('chat.close_sidebar')}
            className="ml-2 hidden h-11 w-11 items-center justify-center rounded-xl text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-800 transition md:flex"
          >
            <PanelLeftClose className="w-4 h-4" />
          </button>
        </div>

        {/* Sessions list */}
        <div className="flex-1 overflow-y-auto p-2.5 space-y-1">
          <div className="px-2 py-1 text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex items-center justify-between">
            <span>{t('chat.recent_history')}</span>
            {isSessionLoading && <LoadingSpinner size="sm" />}
          </div>

          {sessions.length === 0 ? (
            <div className="px-3 py-6 text-center text-xs text-slate-400 dark:text-slate-500">
              {t('chat.no_history')}
            </div>
          ) : (
            sessions.map((s) => {
              const isActive = activeSessionId === s.id;
              const isEditing = editingSessionId === s.id;

              return (
                <div
                  key={s.id}
                  onClick={() => {
                    if (!isEditing) {
                      selectSession(s.id);
                      setIsDrawerOpen(false);
                    }
                  }}
                  className={`group relative flex items-center justify-between px-3 py-2 rounded-xl text-xs cursor-pointer transition ${
                    isActive
                      ? 'bg-slate-200/80 dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-medium'
                      : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800/50 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  {isEditing ? (
                    <div className="flex items-center gap-1 w-full" onClick={(e) => e.stopPropagation()}>
                      <input
                        type="text"
                        value={editTitle}
                        onChange={(e) => setEditTitle(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            if (editTitle.trim()) {
                              renameSession(s.id, editTitle.trim());
                            }
                            setEditingSessionId(null);
                          } else if (e.key === 'Escape') {
                            setEditingSessionId(null);
                          }
                        }}
                        className="flex-1 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg px-2 py-1 text-xs text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        autoFocus
                      />
                      <button
                        onClick={() => {
                          if (editTitle.trim()) {
                            renameSession(s.id, editTitle.trim());
                          }
                          setEditingSessionId(null);
                        }}
                        className="p-1.5 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-emerald-600 dark:text-emerald-400"
                      >
                        <Check className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => setEditingSessionId(null)}
                        className="p-1.5 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-400"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  ) : (
                    <>
                      <span className="truncate pr-4 flex-1">{s.title || t('chat.new_chat')}</span>
                      <div className="flex items-center gap-0.5 shrink-0 transition md:opacity-0 md:group-hover:opacity-100 md:focus-within:opacity-100">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setEditingSessionId(s.id);
                            setEditTitle(s.title || '');
                          }}
                          title={t('chat.rename_chat')}
                          className="p-1.5 rounded-lg hover:text-indigo-600 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 text-slate-400 transition"
                        >
                          <Edit3 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            deleteSession(s.id);
                          }}
                          title={t('chat.delete_chat')}
                          className="p-1.5 rounded-lg hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 text-slate-400 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </>
                  )}
                </div>
              );
            })
          )}

        </div>

        {/* Guest Mode Notice */}
        {!isAuthenticated && (
          <div className="mx-2.5 mb-2 p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-xs">
            <div className="font-semibold text-amber-600 dark:text-amber-400">
              {t('chat.guest_mode')}
            </div>
            <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400 leading-tight">
              {t('chat.guest_hint')}
            </p>
            <Link
              to="/login"
              className="mt-2 inline-flex items-center gap-1 text-[11px] font-semibold text-indigo-600 dark:text-indigo-400 hover:underline"
            >
              <LogIn className="w-3 h-3" />
              <span>{t('chat.login')}</span>
            </Link>
          </div>
        )}

        {/* Sidebar Footer */}
        <div className="p-3 border-t border-slate-200/60 dark:border-slate-800/80">
          {isAuthenticated && user ? (
            <div className="space-y-1">
              {canManage && (
                <Link
                  to="/admin"
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 transition"
                >
                  <LayoutDashboard className="w-4 h-4" />
                  <span>{t('chat.admin_portal')}</span>
                </Link>
              )}
              <div className="flex items-center justify-between p-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800">
                <Link to="/profile" className="flex items-center gap-2.5 min-w-0 flex-1">
                  <div className="w-7 h-7 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 text-white flex items-center justify-center text-xs font-bold uppercase shrink-0">
                    {user.avatar_url ? (
                      <img src={user.avatar_url} alt="" className="w-full h-full rounded-full object-cover" />
                    ) : (
                      user.full_name?.charAt(0) || <UserIcon className="w-3.5 h-3.5" />
                    )}
                  </div>
                  <div className="truncate text-left">
                    <div className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate">
                      {user.full_name}
                    </div>
                    <div className="text-[10px] text-slate-400 truncate">{user.email}</div>
                  </div>
                </Link>
                <button
                  onClick={() => logout()}
                  title={t('chat.logout')}
                  className="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg hover:bg-rose-50 dark:hover:bg-rose-950/40 transition"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            </div>
          ) : (
            <div className="px-3 py-1.5 text-center">
              <span className="text-[11px] font-medium text-slate-400 dark:text-slate-500">
                {t('chat.advisor_name')}
              </span>
            </div>
          )}
        </div>
      </aside>

      {/* Main Chat Workspace */}
      <div className="flex-1 flex flex-col min-w-0 h-full relative">
        {/* Top Header Bar */}
        <header className="h-14 px-3 sm:px-4 lg:px-6 border-b border-slate-100 dark:border-slate-800/80 flex items-center justify-between gap-2 shrink-0 bg-white/80 dark:bg-[#131314]/80 backdrop-blur-md z-20">
          <div className="flex min-w-0 flex-1 items-center gap-1.5 sm:gap-3">
            <button
              onClick={() => setIsDrawerOpen(true)}
              title={t('chat.open_sidebar')}
              aria-label={t('chat.open_sidebar')}
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition md:hidden"
            >
              <PanelLeft className="w-5 h-5" />
            </button>

            {!isSidebarOpen && (
              <button
                onClick={toggleSidebar}
                title={t('chat.open_sidebar')}
                className="hidden h-11 w-11 items-center justify-center rounded-xl text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition md:flex"
              >
                <PanelLeft className="w-5 h-5" />
              </button>
            )}

            <div className="relative min-w-0 max-w-full" ref={modelDropdownRef}>
              <button
                onClick={() => setIsModelDropdownOpen(!isModelDropdownOpen)}
                className="flex min-w-0 max-w-full items-center gap-2 px-2.5 sm:px-3 py-2 sm:py-1.5 rounded-full hover:bg-slate-100 dark:hover:bg-slate-800/80 text-xs sm:text-sm font-semibold transition border border-slate-200/60 dark:border-slate-800 text-slate-800 dark:text-slate-200"
              >
                <div className="w-5 h-5 rounded-full bg-gradient-to-tr from-blue-500 via-indigo-500 to-purple-600 text-white flex items-center justify-center shrink-0">
                  <Sparkles className="w-3 h-3" />
                </div>
                <span className="truncate">{selectedModel ? selectedModel.name : t('chat.advisor_name')}</span>
                <ChevronDown className="w-3.5 h-3.5 text-slate-400 shrink-0" />
              </button>

              {isModelDropdownOpen && (
                <div className="absolute left-0 mt-2 w-80 max-w-[calc(100vw_-_2.5rem)] rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl p-2 z-50 animate-in fade-in zoom-in-95 duration-100">
                  <div className="px-3 py-2 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                    {t('chat.select_model')}
                  </div>
                  <div className="space-y-1 max-h-72 overflow-y-auto">
                    {models.map((m) => (
                      <button
                        key={m.id}
                        onClick={() => {
                          setSelectedModel(m);
                          setIsModelDropdownOpen(false);
                        }}
                        className={`w-full text-left px-3 py-2.5 rounded-xl text-sm transition flex flex-col gap-0.5 ${
                          selectedModel?.id === m.id
                            ? 'bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 font-medium'
                            : 'hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-xs sm:text-sm">{m.name}</span>
                          {m.supports_vision && (
                            <span className="px-1.5 py-0.5 rounded text-[10px] bg-emerald-50 dark:bg-emerald-950 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                              {t('chat.model_supports_vision')}
                            </span>
                          )}
                        </div>
                        <span className="text-[11px] text-slate-400 font-mono">
                          {m.provider_name} • {t('chat.model_context', { k: (m.context_length / 1024).toFixed(0) })}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-1.5 sm:gap-3">

            {isAuthenticated && user && <NotificationBell />}

            <LanguageSwitcher className="shrink-0" />

            {isAuthenticated && user ? (
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                  className="flex h-11 w-11 items-center justify-center rounded-full border border-slate-200/80 dark:border-slate-800 hover:bg-slate-100 dark:hover:bg-slate-800 transition sm:h-auto sm:w-auto sm:justify-start sm:gap-2 sm:p-1 sm:pl-2.5"
                >
                  <span className="text-xs font-semibold text-slate-700 dark:text-slate-300 hidden sm:inline">
                    {user.full_name?.split(' ').pop()}
                  </span>
                  <div className="w-7 h-7 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 text-white flex items-center justify-center font-bold text-xs uppercase">
                    {user.avatar_url ? (
                      <img src={user.avatar_url} alt="" className="w-full h-full rounded-full object-cover" />
                    ) : (
                      user.full_name?.charAt(0) || <UserIcon className="w-3.5 h-3.5" />
                    )}
                  </div>
                </button>

                {isUserMenuOpen && (
                  <div className="absolute right-0 mt-2 w-56 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl p-2 z-50 animate-in fade-in zoom-in-95 duration-100">
                    <div className="px-3 py-2 border-b border-slate-100 dark:border-slate-800">
                      <div className="text-xs font-bold text-slate-900 dark:text-slate-100 truncate">
                        {user.full_name}
                      </div>
                      <div className="text-[10px] text-slate-400 truncate">{user.email}</div>
                    </div>

                    <div className="p-1 space-y-1">
                      {canManage && (
                        <button
                          onClick={() => {
                            setIsUserMenuOpen(false);
                            navigate('/admin');
                          }}
                          className="w-full text-left px-3 py-2 rounded-xl text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center gap-2 transition"
                        >
                          <LayoutDashboard className="w-4 h-4 text-indigo-500" />
                          <span>{t('chat.admin_portal')}</span>
                        </button>
                      )}
                      <button
                        onClick={() => {
                          setIsUserMenuOpen(false);
                          navigate('/profile');
                        }}
                        className="w-full text-left px-3 py-2 rounded-xl text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center gap-2 transition"
                      >
                        <UserIcon className="w-4 h-4 text-slate-400" />
                        <span>{t('chat.my_profile')}</span>
                      </button>
                      <button
                        onClick={() => {
                          setIsUserMenuOpen(false);
                          logout();
                        }}
                        className="w-full text-left px-3 py-2 rounded-xl text-xs font-medium text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 flex items-center gap-2 transition"
                      >
                        <LogOut className="w-4 h-4 text-rose-500" />
                        <span>{t('chat.logout')}</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Link
                to="/login"
                title={t('chat.login')}
                aria-label={t('chat.login')}
                className="flex h-11 w-11 items-center justify-center rounded-full bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm transition sm:h-auto sm:w-auto sm:gap-1.5 sm:px-4 sm:py-2"
              >
                <LogIn className="w-4 h-4" />
                <span className="hidden sm:inline font-semibold text-xs">{t('chat.login')}</span>
              </Link>
            )}
          </div>
        </header>

        {/* Conversation Viewport */}
        <div className="flex-1 overflow-y-auto px-3 py-4 sm:px-4 sm:py-6 scroll-smooth">
          {!hasMessages ? (
            /* Google Gemini Welcome Hero */
            <div className="max-w-2xl mx-auto flex flex-col items-center justify-center min-h-[50vh] sm:min-h-[65vh] text-center px-2 sm:px-4 animate-in fade-in duration-300">
              <div className="relative mb-6">
                <div className="w-16 h-16 rounded-3xl bg-gradient-to-tr from-blue-500 via-indigo-500 to-purple-600 flex items-center justify-center text-white shadow-xl shadow-indigo-500/25">
                  <Sparkles className="w-9 h-9" />
                </div>
                <div className="absolute -inset-1 rounded-3xl bg-gradient-to-tr from-blue-500 to-purple-600 opacity-20 blur-lg pointer-events-none" />
              </div>

              <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-slate-100 mb-2">
                {t('chat.hero_greeting', { name: user ? `, ${user.full_name}` : '' })}
              </h1>
              <p className="text-sm sm:text-base text-slate-500 dark:text-slate-400 max-w-lg mb-8 leading-relaxed">
                {t('chat.hero_subtitle')}
              </p>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full text-left">
                {suggestions.map((s, idx) => {
                  const Icon = s.icon;
                  return (
                    <button
                      key={idx}
                      onClick={() => handleExecuteSend(s.desc)}
                      className="p-4 rounded-2xl border border-slate-200/80 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/40 hover:bg-slate-100 dark:hover:bg-slate-800/80 hover:border-indigo-300 dark:hover:border-indigo-700 transition flex items-start gap-3 text-left group"
                    >
                      <div className={`p-2.5 rounded-xl shrink-0 border ${s.color}`}>
                        <Icon className="w-4 h-4" />
                      </div>
                      <div className="min-w-0">
                        <div className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition mb-0.5">
                          {s.title}
                        </div>
                        <div className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-2 leading-relaxed">
                          {s.desc}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          ) : (
            /* Message Thread */
            <div className="max-w-3xl mx-auto space-y-6 pb-6">
              {messages
                .filter((m) => m.id !== 'welcome')
                .map((m) => {
                  const isUser = m.sender === 'user';
                  const assistantMessages = messages.filter((msg) => msg.sender === 'assistant');
                  const assistantIndex = assistantMessages.findIndex((msg) => msg.id === m.id);
                  const questionsPassed = assistantIndex >= 0 ? assistantMessages.length - 1 - assistantIndex : 0;
                  const isEscalated = submittedEscalateIds.has(m.id);
                  const isReasoningActive = isLoading && messages[messages.length - 1]?.id === m.id && !m.text && Boolean(m.reasoning);
                  const reasoningWordCount = m.reasoning ? m.reasoning.trim().split(/\s+/).filter(Boolean).length : 0;
                  return (
                    <div
                      key={m.id}
                      className={`flex gap-2.5 sm:gap-4 ${isUser ? 'justify-end' : 'justify-start'}`}
                    >
                      {!isUser && (
                        <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-blue-500 via-indigo-500 to-purple-600 flex items-center justify-center text-white shrink-0 mt-0.5 shadow-sm">
                          <Sparkles className="w-4 h-4" />
                        </div>
                      )}

                      <div className={`flex flex-col min-w-0 max-w-[85%] sm:max-w-[78%] ${isUser ? 'items-end' : 'items-start'}`}>
                        {isUser ? (
                          <div className="px-4 py-3 rounded-2xl rounded-tr-xs bg-indigo-600 text-white shadow-xs">
                            {m.images && m.images.length > 0 && (
                              <div className="flex flex-wrap gap-2 mb-2">
                                {m.images.map((img, i) => (
                                  <img
                                    key={i}
                                    src={img}
                                    alt={t('chat.attach_image_btn')}
                                    className="max-h-40 max-w-full rounded-xl object-contain bg-black/20"
                                  />
                                ))}
                              </div>
                            )}
                            <div className="text-sm whitespace-pre-wrap break-words leading-relaxed">{m.text}</div>
                          </div>
                        ) : (
                          <div className="w-full space-y-2">
                            {m.toolCalls && m.toolCalls.length > 0 && (
                              <div className="flex flex-col gap-1.5 mb-2">
                                {m.toolCalls.map((tc, idx) => (
                                  <div
                                    key={idx}
                                    className="inline-flex items-center gap-2 px-2.5 py-1 rounded-lg bg-indigo-50/70 dark:bg-indigo-950/40 border border-indigo-200/60 dark:border-indigo-800/50 text-xs text-indigo-700 dark:text-indigo-300 w-fit"
                                  >
                                    {tc.status === 'running' ? (
                                      <RefreshCw className="w-3.5 h-3.5 animate-spin text-indigo-500 shrink-0" />
                                    ) : (
                                      <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                                    )}
                                    <span className="font-medium">
                                      {tc.name === 'lookup_tuition_fee'
                                        ? tc.status === 'running'
                                          ? t('chat.tool_tuition_searching')
                                          : t('chat.tool_tuition_completed')
                                        : tc.status === 'running'
                                        ? t('chat.tool_searching')
                                        : t('chat.tool_search_completed')}
                                    </span>
                                  </div>
                                ))}
                              </div>
                            )}

                            {m.reasoning && (
                              <div className="mb-2 rounded-xl border border-slate-200/80 dark:border-slate-800 bg-slate-50/60 dark:bg-slate-900/40 overflow-hidden text-xs">
                                <button
                                  type="button"
                                  onClick={() => toggleReasoning(m.id)}
                                  className="w-full px-3 py-2 flex items-center justify-between text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100/50 dark:hover:bg-slate-800/40 transition select-none"
                                >
                                  <div className="flex items-center gap-2 font-medium">
                                    <Brain className={`w-3.5 h-3.5 ${isReasoningActive ? 'text-purple-500 animate-pulse' : 'text-purple-500'}`} />
                                    <span>
                                      {isReasoningActive ? t('chat.thinking_in_progress') : t('chat.thinking_process')}
                                    </span>
                                    {isReasoningActive ? (
                                      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-purple-100 dark:bg-purple-950/60 text-purple-600 dark:text-purple-300 animate-pulse">
                                        <span className="w-1.5 h-1.5 rounded-full bg-purple-500 animate-ping" />
                                        <span>{t('chat.reasoning_streaming')}</span>
                                      </span>
                                    ) : (
                                      <span className="text-[10px] text-slate-400 font-mono">
                                        ({reasoningWordCount} {t('chat.words')})
                                      </span>
                                    )}
                                  </div>
                                  <ChevronDown
                                    className={`w-3.5 h-3.5 transition-transform duration-200 ${
                                      expandedReasoning[m.id] ? 'rotate-180' : ''
                                    }`}
                                  />
                                </button>
                                {expandedReasoning[m.id] && (
                                  <div
                                    ref={isReasoningActive ? reasoningScrollRef : undefined}
                                    className="px-3 py-2 border-t border-slate-200/60 dark:border-slate-800/60 text-slate-600 dark:text-slate-400 whitespace-pre-wrap font-mono text-[11px] leading-relaxed max-h-60 overflow-y-auto bg-white/40 dark:bg-black/20"
                                  >
                                    {m.reasoning}
                                    {isReasoningActive && (
                                      <span className="inline-block w-1.5 h-3 ml-0.5 bg-purple-500 animate-pulse align-middle" />
                                    )}
                                  </div>
                                )}
                              </div>
                            )}

                            <div className="px-1 py-0.5">
                              {m.text === '' && isLoading && !m.reasoning ? (
                                <div className="flex items-center gap-2 text-xs text-slate-400 py-1">
                                  <LoadingSpinner size="sm" />
                                  <span>{t('chat.thinking')}</span>
                                </div>
                              ) : m.text === '' && isLoading && m.reasoning ? (
                                <div className="flex items-center gap-2 text-xs text-slate-400 py-1">
                                  <span className="w-1.5 h-1.5 rounded-full bg-purple-500 animate-ping" />
                                  <span className="text-slate-400 dark:text-slate-500 text-[11px] italic">
                                    {t('chat.generating_response')}
                                  </span>
                                </div>
                              ) : (
                                <FormattedMessageContent
                                  content={m.text}
                                  isStreaming={isLoading && !isUser && messages[messages.length - 1]?.id === m.id}
                                />
                              )}
                            </div>

                            {/* RAG Sources Citations */}
                            {m.sources && m.sources.length > 0 && (
                              <div className="mt-3 pt-3 border-t border-slate-100 dark:border-slate-800">
                                <div className="text-[11px] font-semibold text-slate-400 flex items-center gap-1.5 mb-2 uppercase tracking-wider">
                                  <FileText className="w-3.5 h-3.5" />
                                  <span>{t('chat.reference_docs', { count: m.sources.length })}</span>
                                </div>
                                <div className="flex flex-wrap gap-2">
                                  {m.sources.map((s, idx) => (
                                    <div
                                      key={idx}
                                      className="px-2.5 py-1 rounded-xl bg-slate-100 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700/60 text-[11px] text-slate-600 dark:text-slate-300 flex items-center gap-1.5"
                                    >
                                      <span className="font-semibold text-indigo-600 dark:text-indigo-400">
                                        [{idx + 1}]
                                      </span>
                                      <span className="truncate max-w-[200px]">{s}</span>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            )}

                            {/* Escalation Suggestion Banner */}
                            {shouldSuggestEscalation(m) &&
                              (!isLoading || messages[messages.length - 1]?.id !== m.id) &&
                              questionsPassed < 2 && (
                                <div className="mt-3.5 rounded-2xl border border-indigo-200/80 dark:border-indigo-900/60 bg-gradient-to-r from-indigo-50/90 via-purple-50/50 to-indigo-50/90 dark:from-indigo-950/40 dark:via-purple-950/20 dark:to-indigo-950/40 p-3.5 shadow-xs">
                                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                                    <div className="flex items-start gap-2.5">
                                      <div className="p-2 rounded-xl bg-indigo-600/10 dark:bg-indigo-500/20 text-indigo-600 dark:text-indigo-400 shrink-0 mt-0.5">
                                        <GraduationCap className="w-4 h-4" />
                                      </div>
                                      <div>
                                        <div className="text-xs font-semibold text-slate-800 dark:text-slate-200">
                                          {t('inquiries.suggestion_card_title')}
                                        </div>
                                        <div className="text-[11px] text-slate-600 dark:text-slate-400 mt-0.5 leading-relaxed">
                                          {t('inquiries.suggestion_card_desc')}
                                        </div>
                                      </div>
                                    </div>
                                    <button
                                      type="button"
                                      onClick={() => handleOpenEscalateForMessage(m)}
                                      disabled={isEscalated || questionsPassed >= 1}
                                      className={`w-full sm:w-auto inline-flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-xl text-xs font-semibold shadow-sm transition shrink-0 ${
                                        isEscalated
                                          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 cursor-not-allowed'
                                          : questionsPassed >= 1
                                          ? 'bg-slate-200/60 dark:bg-slate-800 text-slate-400 dark:text-slate-500 border border-slate-300/40 dark:border-slate-700/40 cursor-not-allowed opacity-60'
                                          : 'bg-indigo-600 hover:bg-indigo-500 text-white shadow-indigo-600/20 cursor-pointer'
                                      }`}
                                    >
                                      {isEscalated ? (
                                        <>
                                          <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                                          <span>{t('inquiries.btn_escalated')}</span>
                                        </>
                                      ) : questionsPassed >= 1 ? (
                                        <span>{t('inquiries.btn_outdated')}</span>
                                      ) : (
                                        <>
                                          <GraduationCap className="w-3.5 h-3.5" />
                                          <span>{t('inquiries.btn_escalate')}</span>
                                        </>
                                      )}
                                    </button>
                                  </div>
                                </div>
                              )}

                            {/* Assistant Action Toolbar */}
                            {m.text !== '' && (
                              <div className="flex items-center gap-1 pt-1 text-slate-400 text-xs">
                                <button
                                  onClick={() => handleCopyMessage(m.id, m.text)}
                                  title={t('chat.copy')}
                                  className="p-1.5 rounded-lg hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition flex items-center gap-1"
                                >
                                  {copiedId === m.id ? (
                                    <>
                                      <Check className="w-3.5 h-3.5 text-emerald-500" />
                                      <span className="text-[11px] text-emerald-500">{t('chat.copied')}</span>
                                    </>
                                  ) : (
                                    <Copy className="w-3.5 h-3.5" />
                                  )}
                                </button>

                                <button
                                  onClick={() => handleFeedback(m.id, 'up')}
                                  title={t('chat.helpful')}
                                  className={`p-1.5 rounded-lg transition ${
                                    feedbackMap[m.id] === 'up'
                                      ? 'text-indigo-600 bg-indigo-50 dark:bg-indigo-950'
                                      : 'hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                                  }`}
                                >
                                  <ThumbsUp className="w-3.5 h-3.5" />
                                </button>

                                <button
                                  onClick={() => handleFeedback(m.id, 'down')}
                                  title={t('chat.not_helpful')}
                                  className={`p-1.5 rounded-lg transition ${
                                    feedbackMap[m.id] === 'down'
                                      ? 'text-rose-600 bg-rose-50 dark:bg-rose-950'
                                      : 'hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                                  }`}
                                >
                                  <ThumbsDown className="w-3.5 h-3.5" />
                                </button>

                                <button
                                  onClick={handleRegenerate}
                                  title={t('chat.regenerate')}
                                  className="p-1.5 rounded-lg hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
                                >
                                  <RefreshCw className="w-3.5 h-3.5" />
                                </button>

                                <button
                                  onClick={() => handleOpenEscalateForMessage(m)}
                                  disabled={isEscalated || questionsPassed >= 2}
                                  title={isEscalated ? t('inquiries.btn_escalated') : t('inquiries.btn_escalate')}
                                  className={`p-1.5 rounded-lg transition flex items-center gap-1 ${
                                    isEscalated
                                      ? 'text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 cursor-not-allowed'
                                      : questionsPassed >= 2
                                      ? 'text-slate-300 dark:text-slate-600 cursor-not-allowed opacity-50'
                                      : 'hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950'
                                  }`}
                                >
                                  {isEscalated ? (
                                    <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                                  ) : (
                                    <GraduationCap className="w-3.5 h-3.5" />
                                  )}
                                </button>

                                {m.timestamp && (
                                  <span className="text-[10px] text-slate-400 ml-2 font-mono">
                                    {m.timestamp}
                                  </span>
                                )}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* Bottom Floating Pill Input Bar (Google Gemini / ChatGPT 2024-2025 style) */}
        <div className="shrink-0 w-full max-w-3xl mx-auto p-3 sm:p-4 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:pb-[max(1rem,env(safe-area-inset-bottom))]">
          <div className="rounded-3xl bg-[#f0f4f9] dark:bg-[#1e1f20] border border-slate-200/80 dark:border-slate-800 shadow-sm focus-within:shadow-md focus-within:border-slate-300 dark:focus-within:border-slate-700 transition p-2.5 sm:p-3">
            {/* Attached Images Preview */}
            {attachedImages.length > 0 && (
              <div className="flex flex-wrap gap-2 mb-2 p-1.5 border-b border-slate-200/50 dark:border-slate-800/60">
                {attachedImages.map((img, i) => (
                  <div key={i} className="relative group rounded-xl overflow-hidden border border-slate-300 dark:border-slate-700">
                    <img src={img} alt={t('chat.attach_image_btn')} className="w-14 h-14 object-cover" />
                    <button
                      type="button"
                      onClick={() => removeAttachedImage(i)}
                      title={t('chat.remove_image')}
                      aria-label={t('chat.remove_image')}
                      className="absolute top-0.5 right-0.5 flex h-7 w-7 items-center justify-center rounded-full bg-slate-900/80 text-white hover:bg-rose-600 transition"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex items-end gap-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={handleImageSelect}
              />

              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                title={t('chat.attach_image_btn')}
                aria-label={t('chat.attach_image_btn')}
                className="flex h-11 w-11 items-center justify-center rounded-full text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-800 transition shrink-0"
              >
                <Paperclip className="w-5 h-5" />
              </button>

              <textarea
                ref={textareaRef}
                rows={1}
                value={input}
                onChange={handleTextareaChange}
                onKeyDown={handleKeyDown}
                placeholder={t('chat.input_placeholder')}
                className="flex-1 min-w-0 max-h-44 py-2 px-1 text-base sm:text-sm bg-transparent border-0 focus:outline-hidden focus:ring-0 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 resize-none leading-relaxed"
              />

              {isLoading ? (
                <button
                  type="button"
                  onClick={handleStopGeneration}
                  title={t('chat.stop_generation')}
                  className="flex h-11 w-11 items-center justify-center rounded-full bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 hover:opacity-90 transition shrink-0 shadow-sm"
                >
                  <Square className="w-5 h-5 fill-current" />
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => handleSend()}
                  disabled={!input.trim() && attachedImages.length === 0}
                  title={t('chat.send_prompt')}
                  className={`flex h-11 w-11 items-center justify-center rounded-full transition shrink-0 shadow-sm ${
                    input.trim() || attachedImages.length > 0
                      ? 'bg-indigo-600 hover:bg-indigo-700 text-white'
                      : 'bg-slate-200 dark:bg-slate-800 text-slate-400 cursor-not-allowed'
                  }`}
                >
                  <ArrowUp className="w-5 h-5" />
                </button>
              )}
            </div>
          </div>

          <div className="text-[11px] text-center text-slate-400 mt-2 px-4">
            {t('chat.disclaimer')}
          </div>
        </div>
      </div>

      {escalateToast && (
        <div className="fixed bottom-6 right-6 z-50 rounded-xl bg-emerald-600 px-5 py-3 text-sm text-white shadow-xl backdrop-blur-md">
          {escalateToast}
        </div>
      )}

      {isEscalateModalOpen && (
        <Modal
          isOpen={isEscalateModalOpen}
          onClose={() => setIsEscalateModalOpen(false)}
          title={t('inquiries.modal_escalate_title')}
          maxWidth="2xl"
        >
          <div className="space-y-4">
            <p className="text-xs text-slate-400 leading-relaxed">
              {t('inquiries.modal_escalate_desc')}
            </p>

            {hasAutoFilled && (
              <div className="flex items-center gap-2 rounded-xl bg-indigo-500/10 border border-indigo-500/30 p-2.5 text-xs text-indigo-400">
                <Sparkles className="w-4 h-4 shrink-0 text-indigo-400" />
                <span>{t('inquiries.auto_filled_hint')}</span>
              </div>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="flex items-center justify-between text-xs font-medium text-slate-300">
                  <span>
                    {t('inquiries.student_name_label')} <span className="text-rose-400">*</span>
                  </span>
                  <span className="text-[10px] text-slate-500">{t('inquiries.required_field')}</span>
                </label>
                <input
                  type="text"
                  value={escalateStudentName}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder={t('inquiries.student_name_placeholder')}
                  className={`mt-1 w-full rounded-xl border bg-slate-900 p-2.5 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    formSubmitted && !escalateStudentName.trim()
                      ? 'border-rose-500/80 ring-1 ring-rose-500/50'
                      : 'border-slate-700'
                  }`}
                />
                {formSubmitted && !escalateStudentName.trim() && (
                  <p className="mt-1 text-[11px] text-rose-400">{t('inquiries.required_field_error')}</p>
                )}
              </div>

              <div>
                <label className="flex items-center justify-between text-xs font-medium text-slate-300">
                  <span>
                    {t('inquiries.student_code_label')} <span className="text-rose-400">*</span>
                  </span>
                  <span className="text-[10px] text-slate-500">{t('inquiries.required_field')}</span>
                </label>
                <input
                  type="text"
                  value={escalateStudentCode}
                  onChange={(e) => handleCodeChange(e.target.value)}
                  placeholder={t('inquiries.student_code_placeholder')}
                  className={`mt-1 w-full rounded-xl border bg-slate-900 p-2.5 text-sm text-slate-200 uppercase font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    formSubmitted && !escalateStudentCode.trim()
                      ? 'border-rose-500/80 ring-1 ring-rose-500/50'
                      : 'border-slate-700'
                  }`}
                />
                {formSubmitted && !escalateStudentCode.trim() && (
                  <p className="mt-1 text-[11px] text-rose-400">{t('inquiries.required_field_error')}</p>
                )}
              </div>

              <div>
                <label className="flex items-center justify-between text-xs font-medium text-slate-300">
                  <span>
                    {t('inquiries.student_class_label')} <span className="text-rose-400">*</span>
                  </span>
                  <span className="text-[10px] text-slate-500">{t('inquiries.required_field')}</span>
                </label>
                <input
                  type="text"
                  value={escalateStudentClass}
                  onChange={(e) => handleClassChange(e.target.value)}
                  placeholder={t('inquiries.student_class_placeholder')}
                  className={`mt-1 w-full rounded-xl border bg-slate-900 p-2.5 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    formSubmitted && !escalateStudentClass.trim()
                      ? 'border-rose-500/80 ring-1 ring-rose-500/50'
                      : 'border-slate-700'
                  }`}
                />
                {formSubmitted && !escalateStudentClass.trim() && (
                  <p className="mt-1 text-[11px] text-rose-400">{t('inquiries.required_field_error')}</p>
                )}
              </div>
            </div>

            <div>
              <label className="flex items-center justify-between text-xs font-medium text-slate-300">
                <span>
                  {t('inquiries.escalate_question_label')} <span className="text-rose-400">*</span>
                </span>
                <span className="text-[10px] text-slate-500">{t('inquiries.required_field')}</span>
              </label>
              <textarea
                rows={6}
                value={escalateQuestion}
                onChange={(e) => {
                  setIsQuestionCustomized(true);
                  setEscalateQuestion(e.target.value);
                }}
                placeholder={t('inquiries.escalate_question_placeholder')}
                className={`mt-1 w-full rounded-xl border bg-slate-900 p-2.5 text-sm text-slate-200 font-normal leading-relaxed focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                  formSubmitted && !escalateQuestion.trim()
                    ? 'border-rose-500/80 ring-1 ring-rose-500/50'
                    : 'border-slate-700'
                }`}
              />
              {formSubmitted && !escalateQuestion.trim() && (
                <p className="mt-1 text-[11px] text-rose-400">{t('inquiries.required_field_error')}</p>
              )}
            </div>

            <div>
              <label className="text-xs font-medium text-slate-300">
                {t('inquiries.escalate_context_label')}
              </label>
              <textarea
                rows={4}
                value={escalateContext}
                onChange={(e) => setEscalateContext(e.target.value)}
                placeholder={t('inquiries.escalate_context_placeholder')}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 p-2.5 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setIsEscalateModalOpen(false)}
                className="rounded-xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800"
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                disabled={
                  !escalateStudentName.trim() ||
                  !escalateStudentCode.trim() ||
                  !escalateStudentClass.trim() ||
                  !escalateQuestion.trim() ||
                  isSubmittingEscalate
                }
                onClick={handleSendEscalate}
                className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-indigo-600/20 hover:bg-indigo-500 disabled:opacity-50"
              >
                <Send className="w-4 h-4" />
                <span>{t('inquiries.escalate_submit')}</span>
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};
