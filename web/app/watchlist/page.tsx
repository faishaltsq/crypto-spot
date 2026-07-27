'use client';

import { useEffect, useMemo, useState } from 'react';
import { Bell, BellOff, ChevronDown, ChevronUp, GripVertical, Plus, Search, Star, Trash2, VolumeX } from 'lucide-react';
import { GlobalHeader } from '@/components/terminal/TerminalHeader';
import { formatCompact, formatPercent, formatPrice, formatTime } from '@/lib/format';
import { addPair, createWatchlist, initialWatchlistState, localWatchlistRepository, reorderPairs, type RiskLevel, type Watchlist, type WatchlistPair, type WatchlistState } from '@/lib/watchlist';
import { useMarketStore } from '@/stores/market';

const signalStates = new Set(['ACTIVE', 'CONFIRMED', 'SETUP']);
const riskOptions: RiskLevel[] = ['LOW', 'MEDIUM', 'HIGH'];
const signalTypes = ['BUY_CONFIRMED', 'BUY_SETUP'];

export default function WatchlistPage() {
  const scanner = useMarketStore((state) => state.scanner);
  const signals = useMarketStore((state) => state.signals);
  const initializeScanner = useMarketStore((state) => state.initializeScanner);
  const initializeSignals = useMarketStore((state) => state.initializeSignals);
  const marketLoading = useMarketStore((state) => state.isLoading);
  const marketError = useMarketStore((state) => state.error);
  const [state, setState] = useState<WatchlistState | null>(null);
  const [storageError, setStorageError] = useState('');
  const [search, setSearch] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [activeOnly, setActiveOnly] = useState(false);
  const [staleOnly, setStaleOnly] = useState(false);
  const [mutedOnly, setMutedOnly] = useState(false);
  const [sortBy, setSortBy] = useState<'position' | 'score' | 'change' | 'volume'>('position');
  const [newName, setNewName] = useState('');
  const [newSymbol, setNewSymbol] = useState('');
  const [selectedSymbol, setSelectedSymbol] = useState('');
  const [notice, setNotice] = useState('');
  const [permission, setPermission] = useState<NotificationPermission | 'unsupported'>('unsupported');

  useEffect(() => {
    try {
      const loaded = localWatchlistRepository.load();
      setState(loaded);
      setSelectedSymbol(loaded.watchlists.find((item) => item.id === loaded.selectedWatchlistId)?.pairs[0]?.symbol ?? '');
    } catch {
      const fallback = initialWatchlistState();
      setState(fallback);
      setSelectedSymbol(fallback.watchlists[0].pairs[0]?.symbol ?? '');
      setStorageError('Local storage unavailable. Changes remain in this browser session only.');
    }
    void initializeScanner();
    void initializeSignals();
    setPermission(typeof Notification === 'undefined' ? 'unsupported' : Notification.permission);
  }, [initializeScanner, initializeSignals]);

  const selectedWatchlist = state?.watchlists.find((item) => item.id === state.selectedWatchlistId) ?? state?.watchlists[0];
  const allTags = useMemo(() => [...new Set(selectedWatchlist?.pairs.flatMap((pair) => pair.tags) ?? [])].sort(), [selectedWatchlist]);
  const rows = useMemo(() => {
    if (!selectedWatchlist) return [];
    return selectedWatchlist.pairs
      .map((preference) => ({ preference, market: scanner[preference.symbol], active: signals.some((signal) => signal.symbol === preference.symbol && signalStates.has(signal.status)) }))
      .filter(({ preference, market, active }) => {
        const query = search.trim().toUpperCase();
        if (query && !preference.symbol.includes(query) && !preference.note.toUpperCase().includes(query) && !preference.tags.some((tag) => tag.toUpperCase().includes(query))) return false;
        if (tagFilter && !preference.tags.includes(tagFilter)) return false;
        if (activeOnly && !active) return false;
        if (staleOnly && market?.dataQualityStatus !== 'STALE') return false;
        return !mutedOnly || preference.isMuted;
      })
      .sort((a, b) => {
        if (a.preference.isPinned !== b.preference.isPinned) return a.preference.isPinned ? -1 : 1;
        if (sortBy === 'score') return (b.market?.ruleScore ?? -1) - (a.market?.ruleScore ?? -1);
        if (sortBy === 'change') return (b.market?.change24hPercent ?? -Infinity) - (a.market?.change24hPercent ?? -Infinity);
        if (sortBy === 'volume') return (b.market?.quoteVolume24h ?? -Infinity) - (a.market?.quoteVolume24h ?? -Infinity);
        return a.preference.position - b.preference.position;
      });
  }, [activeOnly, mutedOnly, scanner, search, selectedWatchlist, signals, sortBy, staleOnly, tagFilter]);
  const selectedPair = selectedWatchlist?.pairs.find((pair) => pair.symbol === selectedSymbol) ?? selectedWatchlist?.pairs[0];
  const activeSignals = signals.filter((signal) => selectedWatchlist?.pairs.some((pair) => pair.symbol === signal.symbol) && signalStates.has(signal.status));
  const recentSignals = signals.filter((signal) => selectedWatchlist?.pairs.some((pair) => pair.symbol === signal.symbol) && !signalStates.has(signal.status)).slice(0, 8);

  function persist(next: WatchlistState) {
    setState(next);
    try {
      localWatchlistRepository.save(next);
      setStorageError('');
    } catch {
      setStorageError('Storage error. Local storage unavailable; changes cannot persist after reload.');
    }
  }
  function updateWatchlist(update: (watchlist: Watchlist) => Watchlist) {
    if (!state || !selectedWatchlist) return;
    persist({ ...state, watchlists: state.watchlists.map((watchlist) => watchlist.id === selectedWatchlist.id ? update(watchlist) : watchlist) });
  }
  function updatePair(symbol: string, update: (pair: WatchlistPair) => WatchlistPair) {
    updateWatchlist((watchlist) => ({ ...watchlist, updatedAt: new Date().toISOString(), pairs: watchlist.pairs.map((pair) => pair.symbol === symbol ? { ...update(pair), updatedAt: new Date().toISOString() } : pair) }));
  }
  function createList() {
    if (!state || !newName.trim()) return;
    const watchlist = createWatchlist(newName);
    persist({ ...state, watchlists: [...state.watchlists, watchlist], selectedWatchlistId: watchlist.id });
    setNewName('');
    setSelectedSymbol('');
  }
  function deleteList() {
    if (!state || !selectedWatchlist || state.watchlists.length === 1 || !window.confirm(`Delete ${selectedWatchlist.name}?`)) return;
    const watchlists = state.watchlists.filter((watchlist) => watchlist.id !== selectedWatchlist.id);
    persist({ ...state, watchlists, selectedWatchlistId: watchlists[0].id });
    setSelectedSymbol(watchlists[0].pairs[0]?.symbol ?? '');
  }
  function addNewPair() {
    if (!newSymbol.trim()) return;
    try {
      updateWatchlist((watchlist) => addPair(watchlist, newSymbol));
      setSelectedSymbol(newSymbol.trim().toUpperCase().replace('/', '_'));
      setNewSymbol('');
    } catch (error) { setNotice(error instanceof Error ? error.message : 'Pair gagal ditambahkan.'); }
  }
  function requestPermission() {
    if (typeof Notification === 'undefined') return;
    void Notification.requestPermission().then(setPermission);
  }
  if (!state || !selectedWatchlist) return <main className="watchlist-page"><GlobalHeader /><div className="watchlist-loading" role="status">Loading watchlist...</div></main>;

  return (
    <main className="watchlist-page">
      <GlobalHeader />
      <div className="watchlist-shell">
        <header className="watchlist-head">
          <div><p className="watchlist-kicker">PERSONAL SIGNAL PREFERENCES</p><h1>Watchlist</h1><p>Pair preferences only. Scanner universe and global signal rules stay unchanged.</p></div>
          <div className="storage-status"><span className="status-dot warning" />Saved locally</div>
        </header>
        {storageError && <div className="watchlist-alert errorBox" role="alert">{storageError}</div>}
        {marketError && <div className="watchlist-alert watchlist-warning" role="status">Backend offline. Saved preferences remain available; market data may be unavailable.</div>}
        {notice && <div className="watchlist-alert watchlist-warning" role="status">{notice}<button className="watchlist-link-button" onClick={() => setNotice('')}>Dismiss</button></div>}

        <section className="watchlist-toolbar" aria-label="Watchlist controls">
          <label>Watchlist<select aria-label="Watchlist selector" value={selectedWatchlist?.id} onChange={(event) => { persist({ ...state, selectedWatchlistId: event.target.value }); setSelectedSymbol(state.watchlists.find((item) => item.id === event.target.value)?.pairs[0]?.symbol ?? ''); }}>{state.watchlists.map((watchlist) => <option key={watchlist.id} value={watchlist.id}>{watchlist.name}</option>)}</select></label>
          <input aria-label="New watchlist name" value={newName} onChange={(event) => setNewName(event.target.value)} placeholder="New watchlist" />
          <button className="watchlist-button" onClick={createList}><Plus size={15} />Create</button>
          <input aria-label="Rename watchlist" value={selectedWatchlist?.name ?? ''} onChange={(event) => updateWatchlist((watchlist) => ({ ...watchlist, name: event.target.value, updatedAt: new Date().toISOString() }))} />
          <button className="watchlist-icon-button" title="Delete watchlist" aria-label="Delete watchlist" disabled={state.watchlists.length === 1} onClick={deleteList}><Trash2 size={16} /></button>
          <div className="watchlist-density" aria-label="Table density"><button className={state.density === 'compact' ? 'active' : ''} onClick={() => persist({ ...state, density: 'compact' })}>Compact</button><button className={state.density === 'comfortable' ? 'active' : ''} onClick={() => persist({ ...state, density: 'comfortable' })}>Comfortable</button></div>
        </section>

        <section className="watchlist-summary" aria-label="Watchlist summary">
          <Summary label="Pairs" value={selectedWatchlist?.pairs.length ?? 0} />
          <Summary label="Pinned" value={selectedWatchlist?.pairs.filter((pair) => pair.isPinned).length ?? 0} />
          <Summary label="Alerts enabled" value={selectedWatchlist?.pairs.filter((pair) => pair.notificationEnabled && !pair.isMuted).length ?? 0} />
          <Summary label="Active signals" value={activeSignals.length} />
        </section>

        <section className="watchlist-panel">
          <div className="watchlist-panel-title"><div><h2>Pairs</h2><p>{marketLoading ? 'Loading market data...' : 'Pinned pairs stay first. Reorder only changes this watchlist.'}</p></div><div className="watchlist-add"><input aria-label="Pair symbol" value={newSymbol} onChange={(event) => setNewSymbol(event.target.value)} onKeyDown={(event) => event.key === 'Enter' && addNewPair()} placeholder="BTC_USDT" /><button className="watchlist-button" onClick={addNewPair}><Plus size={15} />Add pair</button></div></div>
          <div className="watchlist-filters"><label className="watchlist-search"><Search size={15} /><input aria-label="Search watchlist" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search pair, note, tag" /></label><select aria-label="Sort pairs" value={sortBy} onChange={(event) => setSortBy(event.target.value as typeof sortBy)}><option value="position">Manual order</option><option value="score">Signal score</option><option value="change">24h change</option><option value="volume">Volume</option></select><select aria-label="Filter tags" value={tagFilter} onChange={(event) => setTagFilter(event.target.value)}><option value="">All tags</option>{allTags.map((tag) => <option key={tag}>{tag}</option>)}</select><FilterButton active={activeOnly} setActive={setActiveOnly}>Active signal</FilterButton><FilterButton active={staleOnly} setActive={setStaleOnly}>Stale</FilterButton><FilterButton active={mutedOnly} setActive={setMutedOnly}>Muted</FilterButton></div>
          <div className={`watchlist-table-wrap ${state.density}`}><table><thead><tr><th aria-label="Reorder" /><th>Pair</th><th>Tier</th><th>Price</th><th>24h</th><th>Quote volume</th><th>Spread</th><th>Signal score</th><th>Signal state</th><th>Data quality</th><th>Alert</th><th>Note</th><th>Tags</th><th>Last update</th><th aria-label="Actions" /></tr></thead><tbody>{rows.length === 0 ? <tr><td colSpan={15} className="emptyCell">No pairs match these filters.</td></tr> : rows.map(({ preference, market, active }) => <tr key={preference.id} className={selectedPair?.symbol === preference.symbol ? 'selectedRow' : ''} onClick={() => setSelectedSymbol(preference.symbol)}><td><div className="watchlist-reorder"><GripVertical size={15} /><button aria-label={`Move ${preference.symbol} up`} onClick={(event) => { event.stopPropagation(); updateWatchlist((watchlist) => ({ ...watchlist, pairs: reorderPairs(watchlist.pairs, watchlist.pairs.findIndex((item) => item.symbol === preference.symbol), Math.max(0, watchlist.pairs.findIndex((item) => item.symbol === preference.symbol) - 1)) })); }}><ChevronUp size={13} /></button><button aria-label={`Move ${preference.symbol} down`} onClick={(event) => { event.stopPropagation(); const index = selectedWatchlist.pairs.findIndex((item) => item.symbol === preference.symbol); updateWatchlist((watchlist) => ({ ...watchlist, pairs: reorderPairs(watchlist.pairs, index, Math.min(watchlist.pairs.length - 1, index + 1)) })); }}><ChevronDown size={13} /></button></div></td><td><strong>{preference.symbol.replace('_', '/')}</strong>{preference.isMuted && <small className="muted-label">Muted</small>}</td><td>{market?.tier ?? '-'}</td><td>{market ? formatPrice(market.price) : 'Unavailable'}</td><td className={market && market.change24hPercent >= 0 ? 'positiveText' : 'negativeText'}>{market ? formatPercent(market.change24hPercent) : '-'}</td><td>{market ? formatCompact(market.quoteVolume24h) : '-'}</td><td>{market ? `${market.spreadBps.toFixed(1)} bps` : '-'}</td><td>{market?.ruleScore.toFixed(0) ?? '-'}</td><td>{active ? <span className="watchlist-badge positive">Active</span> : market?.status ?? '-'}</td><td>{market ? <span className={`watchlist-badge ${market.dataQualityStatus === 'STALE' ? 'warning' : ''}`}>{market.dataQualityStatus}</span> : <span className="watchlist-badge warning">Unavailable</span>}</td><td>{preference.isMuted ? 'Muted' : preference.notificationEnabled ? 'On' : 'Off'}</td><td className="watchlist-note-cell">{preference.note || '-'}</td><td><div className="watchlist-tags">{preference.tags.map((tag) => <span key={tag}>{tag}</span>)}</div></td><td>{market ? formatTime(market.calculatedAt) : '-'}</td><td><button className={`watchlist-icon-button ${preference.isFavorite ? 'active' : ''}`} title="Favorite pair" aria-label={`Favorite ${preference.symbol}`} onClick={(event) => { event.stopPropagation(); updatePair(preference.symbol, (pair) => ({ ...pair, isFavorite: !pair.isFavorite })); }}><Star size={15} fill={preference.isFavorite ? 'currentColor' : 'none'} /></button></td></tr>)}</tbody></table></div>
        </section>

        <div className="watchlist-lower-grid">
          <section className="watchlist-panel"><div className="watchlist-panel-title"><div><h2>Active signals</h2><p>Global signals matching this watchlist.</p></div></div><SignalList signals={activeSignals} empty="No active signals for this watchlist." /></section>
          <section className="watchlist-panel"><div className="watchlist-panel-title"><div><h2>Recent signals</h2><p>Most recent completed or inactive signals.</p></div></div><SignalList signals={recentSignals} empty="No recent signals for this watchlist." /></section>
        </div>

        {selectedPair && <section className="watchlist-panel watchlist-editor"><div className="watchlist-panel-title"><div><h2>{selectedPair.symbol.replace('_', '/')} preferences</h2><p>Alert policy uses browser local timezone: {Intl.DateTimeFormat().resolvedOptions().timeZone}.</p></div><button className="watchlist-icon-button" title="Remove pair" aria-label="Remove pair" onClick={() => { updateWatchlist((watchlist) => ({ ...watchlist, pairs: watchlist.pairs.filter((pair) => pair.symbol !== selectedPair.symbol).map((pair, position) => ({ ...pair, position })) })); setSelectedSymbol(selectedWatchlist.pairs.find((pair) => pair.symbol !== selectedPair.symbol)?.symbol ?? ''); }}><Trash2 size={16} /></button></div><div className="watchlist-editor-grid"><label>Preferred timeframe<select value={selectedPair.preferredTimeframe} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, preferredTimeframe: event.target.value }))}>{['1m', '5m', '15m', '30m', '1h', '4h'].map((value) => <option key={value}>{value}</option>)}</select></label><label>Minimum score<input type="number" min="0" max="100" value={selectedPair.minimumSignalScore} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, minimumSignalScore: Math.max(0, Math.min(100, Number(event.target.value))) }))} /></label><label>Quiet start<input type="time" value={selectedPair.quietHoursStart} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, quietHoursStart: event.target.value }))} /></label><label>Quiet end<input type="time" value={selectedPair.quietHoursEnd} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, quietHoursEnd: event.target.value }))} /></label><label className="watchlist-toggle"><input type="checkbox" checked={selectedPair.isPinned} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, isPinned: event.target.checked }))} />Pin pair</label><label className="watchlist-toggle"><input type="checkbox" checked={selectedPair.isMuted} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, isMuted: event.target.checked }))} /><VolumeX size={15} />Mute pair</label><label className="watchlist-toggle"><input type="checkbox" checked={selectedPair.notificationEnabled} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, notificationEnabled: event.target.checked }))} /><Bell size={15} />Notification enabled</label><button className="watchlist-button" onClick={requestPermission} disabled={permission === 'unsupported'}>{permission === 'granted' ? <Bell size={15} /> : <BellOff size={15} />}Permission: {permission}</button></div><div className="watchlist-check-groups"><CheckGroup title="Risk filter" values={riskOptions} selected={selectedPair.riskLevels} onChange={(value) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, riskLevels: toggleValue(pair.riskLevels, value as RiskLevel) }))} /><CheckGroup title="Signal type filter" values={signalTypes} selected={selectedPair.signalTypes} onChange={(value) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, signalTypes: toggleValue(pair.signalTypes, value) }))} /></div><label className="watchlist-wide-label">Notes<textarea value={selectedPair.note} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, note: event.target.value }))} placeholder="Personal trade context, invalidation reminder, research link..." /></label><label className="watchlist-wide-label">Pair tags<input value={selectedPair.tags.join(', ')} onChange={(event) => updatePair(selectedPair.symbol, (pair) => ({ ...pair, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean).slice(0, 10) }))} placeholder="momentum, catalyst, review" /></label></section>}
      </div>
    </main>
  );
}

function Summary({ label, value }: { label: string; value: number }) { return <div><span>{label}</span><strong>{value}</strong></div>; }
function FilterButton({ active, setActive, children }: { active: boolean; setActive: (value: boolean) => void; children: string }) { return <button className={`watchlist-filter-button ${active ? 'active' : ''}`} onClick={() => setActive(!active)}>{children}</button>; }
function CheckGroup({ title, values, selected, onChange }: { title: string; values: string[]; selected: string[]; onChange: (value: string) => void }) { return <fieldset><legend>{title}</legend>{values.map((value) => <label key={value}><input type="checkbox" checked={selected.includes(value)} onChange={() => onChange(value)} />{value}</label>)}</fieldset>; }
function SignalList({ signals, empty }: { signals: Array<{ id: string; symbol: string; type: string; status: string; ruleScore: number; primaryTimeframe: string; createdAt: string }>; empty: string }) { return signals.length ? <ul className="watchlist-signal-list">{signals.map((signal) => <li key={signal.id}><strong>{signal.symbol.replace('_', '/')}</strong><span>{signal.type} | {signal.primaryTimeframe}</span><span>{signal.ruleScore.toFixed(0)} | {signal.status}</span></li>)}</ul> : <p className="watchlist-empty">{empty}</p>; }
function toggleValue<T extends string>(values: T[], value: T): T[] { return values.includes(value) ? values.filter((item) => item !== value) : [...values, value]; }
