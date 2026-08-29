#!/usr/bin/env node
// ECC Cost Tracker Hook — logs session token usage
// Saves to .claude/cost-log.json
'use strict';

const fs = require('fs');
const path = require('path');

const logFile = path.join(__dirname, '..', 'cost-log.json');
const entry = {
  timestamp: new Date().toISOString(),
  // OpenCode doesn't expose token counts via hooks — this is a placeholder
  // Real implementation requires OpenCode API access
  session: process.env.OPENCODE_SESSION_ID || 'unknown',
  tokens: 'N/A (requires API instrumentation)',
  estimated_cost: 'N/A'
};

let log = [];
try { log = JSON.parse(fs.readFileSync(logFile, 'utf8')); } catch (e) {}
log.push(entry);
fs.writeFileSync(logFile, JSON.stringify(log, null, 2));
console.log('[Hook] cost-tracker: logged session end');
