// Supply chain advisory sources reference
module.exports = {
  sources: [
    { name: 'NVD', url: 'https://nvd.nist.gov/' },
    { name: 'OSV', url: 'https://osv.dev/' },
    { name: 'GitHub Advisory', url: 'https://github.com/advisories' },
    { name: 'Go VulnDB', url: 'https://pkg.go.dev/vuln/' },
  ],
  goTool: 'govulncheck',
  npmTool: 'npm audit'
};
