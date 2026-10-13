const assert = require('node:assert/strict');
const { parseOBJ } = require('./obj_import.js');

const parsed = parseOBJ([
    'o Site',
    'v 0 0 0',
    'v 1 0 0',
    'v 0 1 0',
    'vt 0 0',
    'vt 1 0',
    'vt 0 1',
    'vn 0 0 1',
    'f 1/1/1 2/2/1 3/3/1',
].join('\n'));

assert.equal(parsed.groups.length, 1);
assert.equal(parsed.groups[0].name, 'Site');
assert.equal(parsed.groups[0].positions.length, 9);
assert.equal(parsed.groups[0].indices.length, 3);
assert.deepEqual(parsed.groups[0].indices, new Uint32Array([0, 1, 2]));
assert.throws(() => parseOBJ('v 0 0'), /vertex/);
assert.throws(() => parseOBJ('v 0 0 0\nf 1 2'), /face/);

console.log('OBJ parser checks passed');
