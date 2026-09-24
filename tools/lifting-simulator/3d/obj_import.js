(function (root, factory) {
    const api = factory();
    if (typeof module === 'object' && module.exports) module.exports = api;
    else root.INTEGIN_OBJ = api;
})(typeof self !== 'undefined' ? self : this, function () {
    const MAX_TEXT = 5 * 1024 * 1024;
    const MAX_VERTICES = 200000;
    const MAX_FACES = 200000;

    function finite(value, label) {
        const number = Number(value);
        if (!Number.isFinite(number)) throw new Error(`${label} must be finite`);
        return number;
    }

    function resolveIndex(raw, length, label) {
        if (!/^-?\d+$/.test(raw)) throw new Error(`${label} index is invalid`);
        const value = Number.parseInt(raw, 10);
        const index = value < 0 ? length + value : value - 1;
        if (index < 0 || index >= length) throw new Error(`${label} index is out of range`);
        return index;
    }

    function parseOBJ(text) {
        if (typeof text !== 'string') throw new Error('OBJ source must be text');
        if (text.length > MAX_TEXT) throw new Error('OBJ source exceeds 5 MiB limit');
        const vertices = [];
        const normals = [];
        const uvs = [];
        const groups = [];
        let current = null;
        let faceCount = 0;
        let vertexCount = 0;

        function group(name) {
            if (!current || current.name !== name) {
                current = { name: name || `Group ${groups.length + 1}`, positions: [], normals: [], uvs: [], indices: [] };
                groups.push(current);
            }
            return current;
        }

        function parseFace(token, groupData) {
            const fields = token.split('/');
            if (fields.length < 1 || fields[0] === '') throw new Error('face vertex is missing');
            const vertex = resolveIndex(fields[0], vertices.length, 'face vertex');
            const normal = fields[2] ? resolveIndex(fields[2], normals.length, 'face normal') : -1;
            const uv = fields[1] ? resolveIndex(fields[1], uvs.length, 'face texture') : -1;
            const point = vertices[vertex];
            const normalPoint = normal >= 0 ? normals[normal] : [0, 0, 0];
            const uvPoint = uv >= 0 ? uvs[uv] : [0, 0];
            groupData.positions.push(point[0], point[1], point[2]);
            groupData.normals.push(normalPoint[0], normalPoint[1], normalPoint[2]);
            groupData.uvs.push(uvPoint[0], uvPoint[1]);
            const index = groupData.positions.length / 3 - 1;
            groupData.indices.push(index);
            return index;
        }

        for (const rawLine of text.split(/\r?\n/)) {
            const line = rawLine.trim();
            if (!line || line.startsWith('#')) continue;
            const fields = line.split(/\s+/);
            const kind = fields[0];
            if (kind === 'v') {
                if (fields.length < 4) throw new Error('vertex requires three coordinates');
                vertices.push([finite(fields[1], 'vertex x'), finite(fields[2], 'vertex y'), finite(fields[3], 'vertex z')]);
                vertexCount += 1;
                if (vertexCount > MAX_VERTICES) throw new Error('OBJ vertex limit exceeded');
                continue;
            }
            if (kind === 'vn') {
                if (fields.length < 4) throw new Error('normal requires three coordinates');
                normals.push([finite(fields[1], 'normal x'), finite(fields[2], 'normal y'), finite(fields[3], 'normal z')]);
                continue;
            }
            if (kind === 'vt') {
                if (fields.length < 3) throw new Error('texture coordinate requires two values');
                uvs.push([finite(fields[1], 'texture u'), finite(fields[2], 'texture v')]);
                continue;
            }
            if (kind === 'o' || kind === 'g') {
                group(fields.slice(1).join(' '));
                continue;
            }
            if (kind === 'f') {
                if (fields.length < 4) throw new Error('face requires at least three vertices');
                if (!current) current = { name: 'Group 1', positions: [], normals: [], uvs: [], indices: [] };
                if (groups[groups.length - 1] !== current) groups.push(current);
                for (const token of fields.slice(1)) parseFace(token, current);
                faceCount += 1;
                if (faceCount > MAX_FACES) throw new Error('OBJ face limit exceeded');
            }
        }

        const populated = groups.filter(item => item.indices.length >= 3).map(item => ({
            name: item.name,
            positions: new Float32Array(item.positions),
            normals: new Float32Array(item.normals),
            uvs: new Float32Array(item.uvs),
            indices: new Uint32Array(item.indices),
        }));
        if (populated.length === 0) throw new Error('OBJ source contains no faces');
        return { groups: populated, vertices: vertexCount, faces: faceCount };
    }

    function createOBJGroup(parsed, THREE) {
        if (!THREE) throw new Error('Three.js is required to create an OBJ group');
        const root = new THREE.Group();
        root.name = 'Imported site reference';
        root.userData.sourceType = 'OBJ_REFERENCE';
        parsed.groups.forEach((item, index) => {
            const geometry = new THREE.BufferGeometry();
            geometry.setAttribute('position', new THREE.Float32BufferAttribute(item.positions, 3));
            geometry.setAttribute('normal', new THREE.Float32BufferAttribute(item.normals, 3));
            geometry.setAttribute('uv', new THREE.Float32BufferAttribute(item.uvs, 2));
            geometry.setIndex(new THREE.BufferAttribute(new Uint32Array(item.indices), 1));
            geometry.computeBoundingSphere();
            const material = new THREE.MeshStandardMaterial({ color: 0x94a3b8, roughness: 0.85, metalness: 0.05, transparent: true, opacity: 0.9 });
            const mesh = new THREE.Mesh(geometry, material);
            mesh.name = item.name;
            mesh.userData = { nodeId: `SITE-OBJ-${index + 1}`, sourceType: 'OBJ_REFERENCE' };
            mesh.castShadow = true;
            mesh.receiveShadow = true;
            root.add(mesh);
        });
        return root;
    }

    return { parseOBJ, createOBJGroup };
});
