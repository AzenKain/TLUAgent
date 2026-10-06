import React, { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Network,
  Plus,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Search,
  ArrowRight,
  Info,
  ZoomIn,
  ZoomOut,
  Maximize2,
  Target,
} from 'lucide-react';
import {
  ragAdminService,
  type KGNodeDTO,
  type KGEdgeDTO,
} from '@/services/ragAdminService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';

function getNodeColor(node: KGNodeDTO): string {
  if (node.status === 'SUPERSEDED') return '#94a3b8';
  switch (node.node_type) {
    case 'REGULATION':
      return '#6366f1';
    case 'CONDITION':
      return '#0284c7';
    case 'ENTITY':
      return '#8b5cf6';
    case 'METRIC':
      return '#10b981';
    default:
      return '#4f46e5';
  }
}

function getEdgeColor(relation: string): string {
  switch (relation) {
    case 'SUPERSEDES':
      return '#f59e0b';
    case 'REQUIRES':
      return '#3b82f6';
    case 'AMENDS':
      return '#a855f7';
    case 'CONTRADICTS':
      return '#ef4444';
    default:
      return '#64748b';
  }
}

interface NodeLayoutItem {
  id: string;
  node: KGNodeDTO;
  x: number;
  y: number;
  r: number;
  degree: number;
}

// KnowledgeGraphPage provides high-performance Canvas exploration and management of institutional relationships.
export const KnowledgeGraphPage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [search, setSearch] = useState('');
  const [nodeTypeFilter, setNodeTypeFilter] = useState('');
  const [relationFilter, setRelationFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [densityLimit, setDensityLimit] = useState<number>(40);
  const [focusNeighborsOnly, setFocusNeighborsOnly] = useState<boolean>(false);

  const [selectedNode, setSelectedNode] = useState<KGNodeDTO | null>(null);
  const [selectedEdge, setSelectedEdge] = useState<KGEdgeDTO | null>(null);

  const [addEdgeModalOpen, setAddEdgeModalOpen] = useState(false);
  const [sourceNodeId, setSourceNodeId] = useState('');
  const [targetNodeId, setTargetNodeId] = useState('');
  const [relationType, setRelationType] = useState('SUPERSEDES');
  const [edgeWeight, setEdgeWeight] = useState(1.0);

  const [deleteEdgeConfirmOpen, setDeleteEdgeConfirmOpen] = useState(false);
  const [deletingEdgeId, setDeletingEdgeId] = useState<string | null>(null);

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'error'>('success');

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToastMessage(message);
    setToastType(type);
    setTimeout(() => setToastMessage(null), 4000);
  };

  const { data: stats } = useQuery({
    queryKey: ['admin-rag', 'stats'],
    queryFn: () => ragAdminService.getStats(),
  });

  const {
    data: graphData,
    isLoading,
    isFetching,
    refetch,
  } = useQuery({
    queryKey: ['admin-rag', 'graph'],
    queryFn: () => ragAdminService.getKnowledgeGraph(),
  });

  const addEdgeMutation = useMutation({
    mutationFn: (payload: {
      source_node_id: string;
      target_node_id: string;
      relation_type: string;
      weight: number;
    }) => ragAdminService.upsertKGEdge(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-rag', 'graph'] });
      queryClient.invalidateQueries({ queryKey: ['admin-rag', 'stats'] });
      setAddEdgeModalOpen(false);
      setSourceNodeId('');
      setTargetNodeId('');
      showToast(t('knowledge_graph.edge_created_success'), 'success');
    },
    onError: (err: any) => {
      showToast(err?.response?.data?.message || t('common.error_occurred'), 'error');
    },
  });

  const deleteEdgeMutation = useMutation({
    mutationFn: (id: string) => ragAdminService.deleteKGEdge(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-rag', 'graph'] });
      queryClient.invalidateQueries({ queryKey: ['admin-rag', 'stats'] });
      setDeleteEdgeConfirmOpen(false);
      setDeletingEdgeId(null);
      if (selectedEdge?.id === deletingEdgeId) {
        setSelectedEdge(null);
      }
      showToast(t('knowledge_graph.edge_deleted_success'), 'success');
    },
    onError: (err: any) => {
      showToast(err?.response?.data?.message || t('common.error_occurred'), 'error');
    },
  });

  const rawNodes = graphData?.nodes;
  const rawEdges = graphData?.edges;

  const nodeDegreeMap = useMemo(() => {
    const degrees = new Map<string, number>();
    (rawEdges || []).forEach((edge) => {
      degrees.set(edge.source_node_id, (degrees.get(edge.source_node_id) || 0) + 1);
      degrees.set(edge.target_node_id, (degrees.get(edge.target_node_id) || 0) + 1);
    });
    return degrees;
  }, [rawEdges]);

  const filteredNodes = useMemo(() => {
    if (!rawNodes) return [];
    let list = rawNodes.filter((node) => {
      if (
        search.trim() &&
        !node.name.toLowerCase().includes(search.toLowerCase()) &&
        !node.id.toLowerCase().includes(search.toLowerCase()) &&
        !(node.code || '').toLowerCase().includes(search.toLowerCase())
      ) {
        return false;
      }
      if (nodeTypeFilter && node.node_type !== nodeTypeFilter) {
        return false;
      }
      if (statusFilter && node.status !== statusFilter) {
        return false;
      }
      return true;
    });

    if (focusNeighborsOnly && selectedNode && rawEdges) {
      const neighborSet = new Set<string>([selectedNode.id]);
      rawEdges.forEach((e) => {
        if (e.source_node_id === selectedNode.id) neighborSet.add(e.target_node_id);
        if (e.target_node_id === selectedNode.id) neighborSet.add(e.source_node_id);
      });
      list = list.filter((n) => neighborSet.has(n.id));
    } else if (densityLimit > 0 && !search.trim()) {
      list.sort((a, b) => (nodeDegreeMap.get(b.id) || 0) - (nodeDegreeMap.get(a.id) || 0));
      const topNodes = list.slice(0, densityLimit);
      if (selectedNode && !topNodes.some((n) => n.id === selectedNode.id)) {
        const found = list.find((n) => n.id === selectedNode.id);
        if (found) topNodes.push(found);
      }
      list = topNodes;
    }

    return list;
  }, [rawNodes, search, nodeTypeFilter, statusFilter, focusNeighborsOnly, selectedNode, rawEdges, densityLimit, nodeDegreeMap]);

  const filteredEdges = useMemo(() => {
    if (!rawEdges) return [];
    const validNodeIDs = new Set(filteredNodes.map((n) => n.id));
    return rawEdges.filter((edge) => {
      if (relationFilter && edge.relation_type !== relationFilter) {
        return false;
      }
      return validNodeIDs.has(edge.source_node_id) && validNodeIDs.has(edge.target_node_id);
    });
  }, [rawEdges, filteredNodes, relationFilter]);

  const layoutData = useMemo(() => {
    const count = filteredNodes.length;
    if (count === 0) return { nodes: [] as NodeLayoutItem[], map: new Map<string, NodeLayoutItem>() };

    const width = 1000;
    const height = 700;
    const centerX = width / 2;
    const centerY = height / 2;

    const items: NodeLayoutItem[] = filteredNodes.map((node, i) => {
      const degree = nodeDegreeMap.get(node.id) || 0;
      const r = Math.min(22, Math.max(12, 11 + Math.sqrt(degree) * 2));
      const angle = (2 * Math.PI * i) / count;
      let dist = 220;
      if (degree > 15) dist = 90 + (i % 3) * 25;
      else if (degree > 5) dist = 180 + (i % 4) * 35;
      else dist = 270 + (i % 5) * 30;

      return {
        id: node.id,
        node,
        x: centerX + dist * Math.cos(angle),
        y: centerY + dist * Math.sin(angle),
        r,
        degree,
      };
    });

    const nodeIndexMap = new Map<string, number>();
    items.forEach((it, idx) => nodeIndexMap.set(it.id, idx));

    const iterations = Math.min(80, Math.max(30, Math.floor(2500 / (count + 1))));
    for (let iter = 0; iter < iterations; iter++) {
      const temp = 0.5 * (1 - iter / iterations);

      for (let i = 0; i < items.length; i++) {
        for (let j = i + 1; j < items.length; j++) {
          const a = items[i];
          const b = items[j];
          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const minDist = a.r + b.r + 40;
          if (dist < minDist) {
            const force = ((minDist - dist) / dist) * 14 * temp;
            a.x -= dx * force;
            a.y -= dy * force;
            b.x += dx * force;
            b.y += dy * force;
          }
        }
      }

      for (const edge of filteredEdges) {
        const iA = nodeIndexMap.get(edge.source_node_id);
        const iB = nodeIndexMap.get(edge.target_node_id);
        if (iA !== undefined && iB !== undefined) {
          const a = items[iA];
          const b = items[iB];
          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const targetDist = 110;
          const force = ((dist - targetDist) / dist) * 0.08 * temp;
          a.x += dx * force;
          a.y += dy * force;
          b.x -= dx * force;
          b.y -= dy * force;
        }
      }

      for (let i = 0; i < items.length; i++) {
        const it = items[i];
        it.x += (centerX - it.x) * 0.015 * temp;
        it.y += (centerY - it.y) * 0.015 * temp;
      }
    }

    const map = new Map<string, NodeLayoutItem>();
    items.forEach((it) => map.set(it.id, it));
    return { nodes: items, map };
  }, [filteredNodes, filteredEdges, nodeDegreeMap]);

  // Canvas State & Interaction
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const [zoom, setZoom] = useState<number>(1.0);
  const [pan, setPan] = useState<{ x: number; y: number }>({ x: 0, y: 0 });
  const [hoveredNode, setHoveredNode] = useState<KGNodeDTO | null>(null);
  const [hoverPos, setHoverPos] = useState<{ x: number; y: number } | null>(null);
  const isDraggingRef = useRef(false);
  const dragStartRef = useRef<{ x: number; y: number }>({ x: 0, y: 0 });
  const panStartRef = useRef<{ x: number; y: number }>({ x: 0, y: 0 });

  const resetView = useCallback(() => {
    if (!containerRef.current) return;
    const { clientWidth, clientHeight } = containerRef.current;
    const centerX = 500;
    const centerY = 350;
    setZoom(1.0);
    setPan({
      x: clientWidth / 2 - centerX,
      y: clientHeight / 2 - centerY,
    });
  }, []);

  useEffect(() => {
    resetView();
  }, [resetView]);

  // High-performance Canvas Render Loop
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    const w = rect.width;
    const h = rect.height;

    if (canvas.width !== Math.floor(w * dpr) || canvas.height !== Math.floor(h * dpr)) {
      canvas.width = Math.floor(w * dpr);
      canvas.height = Math.floor(h * dpr);
    }

    ctx.save();
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);

    // Dark sleek background
    ctx.fillStyle = '#090d16';
    ctx.fillRect(0, 0, w, h);

    // Subtle grid dots
    ctx.fillStyle = 'rgba(51, 65, 85, 0.4)';
    const gridSize = 40 * zoom;
    const offsetX = pan.x % gridSize;
    const offsetY = pan.y % gridSize;
    for (let x = offsetX; x < w; x += gridSize) {
      for (let y = offsetY; y < h; y += gridSize) {
        ctx.beginPath();
        ctx.arc(x, y, 1, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    // World transformation
    ctx.save();
    ctx.translate(pan.x, pan.y);
    ctx.scale(zoom, zoom);

    const { nodes, map } = layoutData;
    const selectedNeighbors = new Set<string>();
    if (selectedNode) {
      selectedNeighbors.add(selectedNode.id);
      filteredEdges.forEach((e) => {
        if (e.source_node_id === selectedNode.id) selectedNeighbors.add(e.target_node_id);
        if (e.target_node_id === selectedNode.id) selectedNeighbors.add(e.source_node_id);
      });
    }

    // Draw Edges
    for (const edge of filteredEdges) {
      const srcNode = map.get(edge.source_node_id);
      const tgtNode = map.get(edge.target_node_id);
      if (!srcNode || !tgtNode) continue;

      const isConnectedToSelected =
        selectedNode && (edge.source_node_id === selectedNode.id || edge.target_node_id === selectedNode.id);
      const isSelectedEdge = selectedEdge?.id === edge.id;
      const isDimmed = selectedNode && !isConnectedToSelected;

      ctx.beginPath();
      ctx.moveTo(srcNode.x, srcNode.y);
      ctx.lineTo(tgtNode.x, tgtNode.y);

      const color = getEdgeColor(edge.relation_type);
      ctx.strokeStyle = isSelectedEdge ? '#ffffff' : color;
      ctx.globalAlpha = isDimmed ? 0.08 : isConnectedToSelected ? 0.95 : 0.45;
      ctx.lineWidth = isSelectedEdge ? 3.5 : isConnectedToSelected ? 2.5 : 1.4;

      if (edge.relation_type === 'SUPERSEDES') {
        ctx.setLineDash([5, 4]);
      } else {
        ctx.setLineDash([]);
      }
      ctx.stroke();

      // Arrowhead
      const angle = Math.atan2(tgtNode.y - srcNode.y, tgtNode.x - srcNode.x);
      const arrowLength = 9;
      const arrowX = tgtNode.x - (tgtNode.r + 3) * Math.cos(angle);
      const arrowY = tgtNode.y - (tgtNode.r + 3) * Math.sin(angle);

      ctx.beginPath();
      ctx.moveTo(arrowX, arrowY);
      ctx.lineTo(
        arrowX - arrowLength * Math.cos(angle - Math.PI / 6),
        arrowY - arrowLength * Math.sin(angle - Math.PI / 6)
      );
      ctx.lineTo(
        arrowX - arrowLength * Math.cos(angle + Math.PI / 6),
        arrowY - arrowLength * Math.sin(angle + Math.PI / 6)
      );
      ctx.closePath();
      ctx.fillStyle = isSelectedEdge ? '#ffffff' : color;
      ctx.fill();

      // Draw relation label if zoomed in or edge selected
      if ((zoom >= 1.0 && !isDimmed) || isConnectedToSelected || isSelectedEdge) {
        const midX = (srcNode.x + tgtNode.x) / 2;
        const midY = (srcNode.y + tgtNode.y) / 2;
        ctx.font = 'bold 9px monospace';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillStyle = isSelectedEdge ? '#ffffff' : '#94a3b8';
        ctx.fillText(edge.relation_type, midX, midY - 6);
      }
    }

    ctx.setLineDash([]);
    ctx.globalAlpha = 1.0;

    // Draw Nodes
    for (const it of nodes) {
      const isSelected = selectedNode?.id === it.id;
      const isHovered = hoveredNode?.id === it.id;
      const isNeighbor = selectedNode && selectedNeighbors.has(it.id);
      const isDimmed = selectedNode && !isNeighbor;

      ctx.globalAlpha = isDimmed ? 0.2 : 1.0;

      // Outer glow for selected or hovered node
      if (isSelected || isHovered) {
        ctx.beginPath();
        ctx.arc(it.x, it.y, it.r + (isSelected ? 8 : 5), 0, Math.PI * 2);
        ctx.fillStyle = isSelected ? 'rgba(99, 102, 241, 0.35)' : 'rgba(255, 255, 255, 0.2)';
        ctx.fill();
      }

      // Main Node Circle
      ctx.beginPath();
      ctx.arc(it.x, it.y, it.r, 0, Math.PI * 2);
      ctx.fillStyle = getNodeColor(it.node);
      ctx.fill();

      // Border
      ctx.lineWidth = isSelected ? 3 : isNeighbor ? 2 : 1.5;
      ctx.strokeStyle = isSelected ? '#ffffff' : isNeighbor ? '#e2e8f0' : '#0f172a';
      ctx.stroke();

      // Node label
      if (zoom >= 0.75 || isSelected || isHovered || isNeighbor) {
        ctx.font = isSelected ? 'bold 11px sans-serif' : '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'top';

        const label = it.node.name.length > 20 ? it.node.name.slice(0, 18) + '...' : it.node.name;
        // Text shadow for legibility
        ctx.shadowColor = 'rgba(0, 0, 0, 0.9)';
        ctx.shadowBlur = 4;
        ctx.fillStyle = isSelected ? '#ffffff' : '#f1f5f9';
        ctx.fillText(label, it.x, it.y + it.r + 4);
        ctx.shadowBlur = 0;
      }
    }

    ctx.restore();
    ctx.restore();
  }, [layoutData, filteredEdges, selectedNode, selectedEdge, hoveredNode, zoom, pan]);

  // Mouse & Touch Controls
  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (e.button !== 0) return;
    isDraggingRef.current = true;
    dragStartRef.current = { x: e.clientX, y: e.clientY };
    panStartRef.current = { ...pan };
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (isDraggingRef.current) {
      const dx = e.clientX - dragStartRef.current.x;
      const dy = e.clientY - dragStartRef.current.y;
      setPan({
        x: panStartRef.current.x + dx,
        y: panStartRef.current.y + dy,
      });
      return;
    }

    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;

    const worldX = (mouseX - pan.x) / zoom;
    const worldY = (mouseY - pan.y) / zoom;

    let found: KGNodeDTO | null = null;
    for (const it of layoutData.nodes) {
      const dx = worldX - it.x;
      const dy = worldY - it.y;
      if (dx * dx + dy * dy <= (it.r + 6) * (it.r + 6)) {
        found = it.node;
        break;
      }
    }

    if (found !== hoveredNode) {
      setHoveredNode(found);
      if (found) {
        setHoverPos({ x: mouseX, y: mouseY });
      } else {
        setHoverPos(null);
      }
    } else if (found) {
      setHoverPos({ x: mouseX, y: mouseY });
    }
  };

  const handleMouseUp = (e: React.MouseEvent<HTMLCanvasElement>) => {
    isDraggingRef.current = false;

    const dx = Math.abs(e.clientX - dragStartRef.current.x);
    const dy = Math.abs(e.clientY - dragStartRef.current.y);

    // If moved less than 4px, treat as click
    if (dx < 4 && dy < 4) {
      if (hoveredNode) {
        setSelectedNode(hoveredNode);
        setSelectedEdge(null);
      } else {
        setSelectedNode(null);
        setSelectedEdge(null);
      }
    }
  };

  const handleWheel = (e: React.WheelEvent<HTMLCanvasElement>) => {
    e.preventDefault();
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;

    const zoomFactor = e.deltaY < 0 ? 1.15 : 0.87;
    const newZoom = Math.min(Math.max(0.25, zoom * zoomFactor), 4.0);

    const worldX = (mouseX - pan.x) / zoom;
    const worldY = (mouseY - pan.y) / zoom;

    setPan({
      x: mouseX - worldX * newZoom,
      y: mouseY - worldY * newZoom,
    });
    setZoom(newZoom);
  };

  const handleZoom = (factor: number) => {
    if (!containerRef.current) return;
    const w = containerRef.current.clientWidth;
    const h = containerRef.current.clientHeight;
    const centerX = w / 2;
    const centerY = h / 2;
    const newZoom = Math.min(Math.max(0.25, zoom * factor), 4.0);
    const worldX = (centerX - pan.x) / zoom;
    const worldY = (centerY - pan.y) / zoom;
    setPan({
      x: centerX - worldX * newZoom,
      y: centerY - worldY * newZoom,
    });
    setZoom(newZoom);
  };

  const handleCreateEdge = (e: React.FormEvent) => {
    e.preventDefault();
    if (!sourceNodeId || !targetNodeId || sourceNodeId === targetNodeId) {
      return;
    }
    addEdgeMutation.mutate({
      source_node_id: sourceNodeId,
      target_node_id: targetNodeId,
      relation_type: relationType,
      weight: Number(edgeWeight),
    });
  };

  return (
    <div className="space-y-6">
      {toastMessage && (
        <div
          className={`fixed bottom-5 right-5 z-50 flex items-center gap-2 rounded-xl px-4 py-3 text-sm font-medium text-white shadow-xl transition-all ${
            toastType === 'success' ? 'bg-emerald-600' : 'bg-rose-600'
          }`}
        >
          {toastType === 'success' ? (
            <CheckCircle2 className="h-5 w-5" />
          ) : (
            <AlertTriangle className="h-5 w-5" />
          )}
          <span>{toastMessage}</span>
        </div>
      )}

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {t('knowledge_graph.title')}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('knowledge_graph.subtitle')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => refetch()}
            className="inline-flex items-center rounded-lg border border-slate-200 bg-white p-2.5 text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
          >
            <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => setAddEdgeModalOpen(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-xs hover:bg-indigo-700 transition"
          >
            <Plus className="h-4 w-4" />
            {t('knowledge_graph.btn_add_edge')}
          </button>
        </div>
      </div>

      {/* Stats Summary Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-indigo-50 p-2.5 text-indigo-600 dark:bg-indigo-950/50 dark:text-indigo-400">
              <Network className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('knowledge_graph.stat_total_nodes')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.total_nodes ?? graphData?.nodes.length ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-sky-50 p-2.5 text-sky-600 dark:bg-sky-950/50 dark:text-sky-400">
              <Network className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('knowledge_graph.stat_total_edges')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.total_edges ?? graphData?.edges.length ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-emerald-50 p-2.5 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('knowledge_graph.stat_active_edges')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {graphData?.edges.filter((e) => e.status === 'ACTIVE').length ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-amber-50 p-2.5 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('knowledge_graph.stat_superseded_nodes')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {graphData?.nodes.filter((n) => n.status === 'SUPERSEDED').length ?? 0}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Filter and Density Toolbar */}
      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('common.search')}
              className="w-full rounded-lg border border-slate-200 bg-white py-2 pl-9 pr-3 text-sm text-slate-800 placeholder-slate-400 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <select
              value={nodeTypeFilter}
              onChange={(e) => setNodeTypeFilter(e.target.value)}
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="" className="bg-slate-900 text-slate-100">{t('knowledge_graph.filter_node_type')}</option>
              <option value="REGULATION" className="bg-slate-900 text-slate-100">REGULATION</option>
              <option value="CONDITION" className="bg-slate-900 text-slate-100">CONDITION</option>
              <option value="ENTITY" className="bg-slate-900 text-slate-100">ENTITY</option>
              <option value="METRIC" className="bg-slate-900 text-slate-100">METRIC</option>
            </select>
          </div>

          <div>
            <select
              value={relationFilter}
              onChange={(e) => setRelationFilter(e.target.value)}
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="" className="bg-slate-900 text-slate-100">{t('knowledge_graph.filter_relation_type')}</option>
              <option value="SUPERSEDES" className="bg-slate-900 text-slate-100">SUPERSEDES</option>
              <option value="REQUIRES" className="bg-slate-900 text-slate-100">REQUIRES</option>
              <option value="AMENDS" className="bg-slate-900 text-slate-100">AMENDS</option>
              <option value="CONTRADICTS" className="bg-slate-900 text-slate-100">CONTRADICTS</option>
            </select>
          </div>

          <div>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="" className="bg-slate-900 text-slate-100">{t('knowledge_graph.filter_status')}</option>
              <option value="ACTIVE" className="bg-slate-900 text-slate-100">{t('documents.status_active')}</option>
              <option value="SUPERSEDED" className="bg-slate-900 text-slate-100">{t('documents.status_superseded')}</option>
            </select>
          </div>

          <div>
            <select
              value={densityLimit}
              onChange={(e) => setDensityLimit(Number(e.target.value))}
              className="w-full rounded-lg border border-indigo-500/50 bg-slate-800 px-3 py-2 text-sm font-medium text-slate-100 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value={40} className="bg-slate-900 text-slate-100">{t('knowledge_graph.density_top_40')}</option>
              <option value={80} className="bg-slate-900 text-slate-100">{t('knowledge_graph.density_top_80')}</option>
              <option value={150} className="bg-slate-900 text-slate-100">{t('knowledge_graph.density_top_150')}</option>
              <option value={0} className="bg-slate-900 text-slate-100">{t('knowledge_graph.density_all')}</option>
            </select>
          </div>
        </div>
      </div>

      {/* Main Canvas Graph & Inspector */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div
          ref={containerRef}
          className="relative h-[620px] overflow-hidden rounded-2xl border border-slate-800 bg-[#090d16] shadow-xl lg:col-span-2 select-none"
        >
          {isLoading ? (
            <div className="flex h-full items-center justify-center">
              <LoadingSpinner size="lg" />
            </div>
          ) : filteredNodes.length === 0 ? (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-slate-500">
              <Network className="h-10 w-10 text-slate-600" />
              <p className="text-sm">{t('knowledge_graph.canvas_empty')}</p>
            </div>
          ) : (
            <>
              <canvas
                ref={canvasRef}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
                onWheel={handleWheel}
                className="h-full w-full cursor-grab active:cursor-grabbing block"
              />

              {/* Floating Zoom & Layout Controls */}
              <div className="absolute top-4 left-4 flex flex-col gap-1.5 rounded-xl border border-slate-800 bg-slate-900/90 p-1.5 shadow-lg backdrop-blur-md">
                <button
                  type="button"
                  title={t('knowledge_graph.zoom_in')}
                  onClick={() => handleZoom(1.25)}
                  className="rounded-lg p-2 text-slate-300 hover:bg-slate-800 hover:text-white transition"
                >
                  <ZoomIn className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  title={t('knowledge_graph.zoom_out')}
                  onClick={() => handleZoom(0.8)}
                  className="rounded-lg p-2 text-slate-300 hover:bg-slate-800 hover:text-white transition"
                >
                  <ZoomOut className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  title={t('knowledge_graph.reset_view')}
                  onClick={resetView}
                  className="rounded-lg p-2 text-slate-300 hover:bg-slate-800 hover:text-white transition"
                >
                  <Maximize2 className="h-4 w-4" />
                </button>
              </div>

              {/* Focus Neighbor Subgraph Toggle */}
              {selectedNode && (
                <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setFocusNeighborsOnly(!focusNeighborsOnly)}
                    className={`flex items-center gap-1.5 rounded-xl border px-3 py-1.5 text-xs font-semibold shadow-md backdrop-blur-md transition ${
                      focusNeighborsOnly
                        ? 'border-indigo-500 bg-indigo-600 text-white'
                        : 'border-slate-700 bg-slate-900/90 text-slate-300 hover:bg-slate-800'
                    }`}
                  >
                    <Target className="h-3.5 w-3.5" />
                    {t('knowledge_graph.focus_only_neighbors')}
                  </button>
                </div>
              )}

              {/* Tooltip on Hover */}
              {hoveredNode && hoverPos && (
                <div
                  className="pointer-events-none absolute z-30 max-w-xs rounded-xl border border-slate-700 bg-slate-900/95 p-3 text-xs text-white shadow-2xl backdrop-blur-md"
                  style={{
                    left: hoverPos.x + 14,
                    top: hoverPos.y + 14,
                    transform: hoverPos.x > 450 ? 'translateX(-110%)' : 'none',
                  }}
                >
                  <div className="font-bold text-indigo-400 leading-tight">{hoveredNode.name}</div>
                  <div className="mt-1 font-mono text-[10px] text-slate-400">{hoveredNode.code || hoveredNode.id}</div>
                  <div className="mt-2 flex items-center gap-2">
                    <span className="rounded bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300">
                      {hoveredNode.node_type}
                    </span>
                    <span className="text-[10px] text-slate-400">
                      {nodeDegreeMap.get(hoveredNode.id) || 0} {t('knowledge_graph.filter_relation_type')}
                    </span>
                  </div>
                </div>
              )}

              {/* Bottom Information Legend & Hint */}
              <div className="absolute bottom-3 left-3 right-3 flex flex-wrap items-center justify-between gap-2 pointer-events-none">
                <div className="flex flex-wrap items-center gap-2 rounded-xl bg-slate-950/85 px-3 py-1.5 text-[11px] text-slate-300 backdrop-blur-md border border-slate-800/80 pointer-events-auto">
                  <div className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-amber-500" />
                    <span>SUPERSEDES</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-blue-500" />
                    <span>REQUIRES</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-purple-500" />
                    <span>AMENDS</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-rose-500" />
                    <span>CONTRADICTS</span>
                  </div>
                </div>

                <div className="rounded-xl bg-slate-950/85 px-3 py-1.5 text-[10px] text-slate-400 backdrop-blur-md border border-slate-800/80 pointer-events-auto">
                  <span>
                    {filteredNodes.length} / {graphData?.nodes.length || 0} nodes • {filteredEdges.length} edges
                  </span>
                  <span className="hidden sm:inline text-slate-600 mx-2">|</span>
                  <span className="hidden sm:inline">{t('knowledge_graph.graph_hint')}</span>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Node & Edge Inspector Panel */}
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-xs dark:border-slate-800 dark:bg-slate-900">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
              <Info className="h-4 w-4 text-indigo-500" />
              {t('knowledge_graph.node_detail_title')}
            </h2>

            {selectedNode ? (
              <div className="mt-4 space-y-3 text-sm">
                <div>
                  <span className="text-xs text-slate-400">{t('knowledge_graph.node_name')}</span>
                  <p className="font-semibold text-slate-900 dark:text-slate-100">
                    {selectedNode.name}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-slate-400">{t('knowledge_graph.node_code')}</span>
                  <p className="font-mono text-xs text-slate-700 dark:text-slate-300">
                    {selectedNode.code || selectedNode.id}
                  </p>
                </div>
                <div className="flex items-center gap-4">
                  <div>
                    <span className="text-xs text-slate-400">{t('knowledge_graph.node_type')}</span>
                    <p className="text-xs font-medium text-slate-700 dark:text-slate-300">
                      {selectedNode.node_type}
                    </p>
                  </div>
                  <div>
                    <span className="text-xs text-slate-400">{t('knowledge_graph.node_status')}</span>
                    <div className="mt-0.5">
                      <Badge variant={selectedNode.status === 'ACTIVE' ? 'success' : 'warning'}>
                        {selectedNode.status}
                      </Badge>
                    </div>
                  </div>
                </div>

                <div className="pt-2 border-t border-slate-100 dark:border-slate-800">
                  <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                    {t('knowledge_graph.connected_edges', {
                      count: (graphData?.edges || []).filter(
                        (e) =>
                          e.source_node_id === selectedNode.id ||
                          e.target_node_id === selectedNode.id
                      ).length,
                    })}
                  </span>
                  <ul className="mt-2 space-y-1.5 max-h-56 overflow-y-auto">
                    {(graphData?.edges || [])
                      .filter(
                        (e) =>
                          e.source_node_id === selectedNode.id ||
                          e.target_node_id === selectedNode.id
                      )
                      .map((edge) => (
                        <li
                          key={edge.id}
                          className="flex items-center justify-between rounded-lg border border-slate-200 p-2 text-xs dark:border-slate-800"
                        >
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span
                              className="font-semibold"
                              style={{ color: getEdgeColor(edge.relation_type) }}
                            >
                              {edge.relation_type}
                            </span>
                            <ArrowRight className="h-3 w-3 text-slate-400 shrink-0" />
                            <span className="truncate max-w-[130px] text-slate-600 dark:text-slate-400">
                              {edge.source_node_id === selectedNode.id
                                ? edge.target_node_id
                                : edge.source_node_id}
                            </span>
                          </div>
                          <button
                            onClick={() => {
                              setDeletingEdgeId(edge.id);
                              setDeleteEdgeConfirmOpen(true);
                            }}
                            className="rounded p-1 text-slate-400 hover:text-rose-600"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </li>
                      ))}
                  </ul>
                </div>
              </div>
            ) : selectedEdge ? (
              <div className="mt-4 space-y-3 text-sm">
                <div>
                  <span className="text-xs text-slate-400">{t('knowledge_graph.relation_type')}</span>
                  <p className="font-semibold text-indigo-600 dark:text-indigo-400">
                    {selectedEdge.relation_type}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-slate-400">{t('knowledge_graph.source_node')}</span>
                  <p className="font-mono text-xs text-slate-700 dark:text-slate-300">
                    {selectedEdge.source_node_id}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-slate-400">{t('knowledge_graph.target_node')}</span>
                  <p className="font-mono text-xs text-slate-700 dark:text-slate-300">
                    {selectedEdge.target_node_id}
                  </p>
                </div>
                <button
                  onClick={() => {
                    setDeletingEdgeId(selectedEdge.id);
                    setDeleteEdgeConfirmOpen(true);
                  }}
                  className="mt-2 w-full rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs font-medium text-rose-600 hover:bg-rose-100 dark:border-rose-900/50 dark:bg-rose-950/30 dark:text-rose-400"
                >
                  {t('knowledge_graph.delete_edge_confirm_title')}
                </button>
              </div>
            ) : (
              <p className="mt-4 text-xs text-slate-400">
                {t('knowledge_graph.click_to_inspect')}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Add Edge Modal */}
      <Modal
        isOpen={addEdgeModalOpen}
        onClose={() => setAddEdgeModalOpen(false)}
        title={t('knowledge_graph.add_edge_title')}
        maxWidth="md"
      >
        <form onSubmit={handleCreateEdge} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('knowledge_graph.source_node')}
            </label>
            <select
              value={sourceNodeId}
              onChange={(e) => setSourceNodeId(e.target.value)}
              required
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="" className="bg-slate-900 text-slate-100">{t('knowledge_graph.select_source_node')}</option>
              {(graphData?.nodes || []).slice(0, 300).map((node) => (
                <option key={node.id} value={node.id} className="bg-slate-900 text-slate-100">
                  {node.name} ({node.code || node.id})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('knowledge_graph.target_node')}
            </label>
            <select
              value={targetNodeId}
              onChange={(e) => setTargetNodeId(e.target.value)}
              required
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="" className="bg-slate-900 text-slate-100">{t('knowledge_graph.select_target_node')}</option>
              {(graphData?.nodes || []).slice(0, 300).map((node) => (
                <option key={node.id} value={node.id} className="bg-slate-900 text-slate-100">
                  {node.name} ({node.code || node.id})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('knowledge_graph.relation_type')}
            </label>
            <select
              value={relationType}
              onChange={(e) => setRelationType(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 [&>option]:bg-slate-900 [&>option]:text-slate-100"
            >
              <option value="SUPERSEDES" className="bg-slate-900 text-slate-100">SUPERSEDES</option>
              <option value="REQUIRES" className="bg-slate-900 text-slate-100">REQUIRES</option>
              <option value="AMENDS" className="bg-slate-900 text-slate-100">AMENDS</option>
              <option value="CONTRADICTS" className="bg-slate-900 text-slate-100">CONTRADICTS</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('knowledge_graph.weight')}
            </label>
            <input
              type="number"
              step="0.1"
              min="0.1"
              max="1.0"
              value={edgeWeight}
              onChange={(e) => setEdgeWeight(Number(e.target.value))}
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setAddEdgeModalOpen(false)}
              className="rounded-lg border border-slate-200 px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={addEdgeMutation.isPending}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {addEdgeMutation.isPending ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Edge Confirmation */}
      <Modal
        isOpen={deleteEdgeConfirmOpen}
        onClose={() => setDeleteEdgeConfirmOpen(false)}
        title={t('knowledge_graph.delete_edge_confirm_title')}
        maxWidth="sm"
      >
        <div className="space-y-4">
          <p className="text-xs text-slate-600 dark:text-slate-300">
            {t('knowledge_graph.delete_edge_confirm_desc')}
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setDeleteEdgeConfirmOpen(false)}
              className="rounded-lg border border-slate-200 px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              disabled={deleteEdgeMutation.isPending}
              onClick={() => deletingEdgeId && deleteEdgeMutation.mutate(deletingEdgeId)}
              className="rounded-lg bg-rose-600 px-4 py-2 text-xs font-medium text-white hover:bg-rose-700 disabled:opacity-50"
            >
              {deleteEdgeMutation.isPending ? t('common.loading') : t('common.delete')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default KnowledgeGraphPage;
