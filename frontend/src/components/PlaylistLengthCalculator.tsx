import { useState } from "react";
import { Play, Clock, Film, Users, Calendar, ExternalLink, Youtube, Loader2, AlertCircle, Github } from "lucide-react";

// Type definitions
interface Video {
    id: string;
    title: string;
    description: string;
    thumbnail: string;
    duration: string;
    duration_sec: number;
}

interface PlaylistResult {
    id: string;
    title: string;
    description: string;
    thumbnail: string;
    videos: Video[];
    total_duration_sec: number;
}

const PlaylistLengthCalculator = () => {
    const [url, setUrl] = useState<string>('');
    const [result, setResult] = useState<PlaylistResult | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [error, setError] = useState<string>('');

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError('');
        setResult(null);

        try {
            const response = await fetch(`http://localhost:8080/api/playlist/analyze`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ youtube_url: url })
            });

            if (!response.ok) {
                throw new Error('Failed to fetch playlist data');
            }

            const data: PlaylistResult = await response.json();
            setResult(data);
        } catch (error) {
            setError('Failed to analyze playlist. Please check the URL and try again.');
            console.error('Error:', error);
        } finally {
            setLoading(false);
        }
    };

    const formatDuration = (seconds: number): string => {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;

        if (hours > 0) {
            return `${hours}h ${minutes}m ${secs}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        } else {
            return `${secs}s`;
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-red-50 via-white to-red-50 flex flex-col">
            {/* Header */}
            <div className="bg-white shadow-sm border-b border-red-100">
                <div className="max-w-6xl mx-auto px-6 py-6">
                    <div className="flex items-center justify-center gap-3">
                        <div className="bg-red-600 p-3 rounded-xl">
                            <Youtube className="w-8 h-8 text-white" />
                        </div>
                        <div className="text-center">
                            <h1 className="text-3xl font-bold text-gray-900">Playlist Length Calculator</h1>
                            <p className="text-gray-600 mt-1">Calculate the total duration of any YouTube playlist</p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="max-w-6xl mx-auto px-6 py-8 flex-1 flex flex-col justify-center">
                {/* Input Section */}
                <div className="bg-white rounded-2xl shadow-lg border border-red-100 p-8 mb-8">
                    <div className="text-center mb-8">
                        <div className="bg-red-100 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                            <Play className="w-8 h-8 text-red-600" />
                        </div>
                        <h2 className="text-2xl font-semibold text-gray-900 mb-2">Analyze Your Playlist</h2>
                        <p className="text-gray-600">Enter a YouTube playlist URL to calculate its total duration</p>
                    </div>

                    <div className="max-w-2xl mx-auto">
                        <div className="relative mb-6">
                            <input
                                type="url"
                                value={url}
                                onChange={(e) => setUrl(e.target.value)}
                                placeholder="https://www.youtube.com/playlist?list=..."
                                className="w-full px-6 py-4 pr-16 border-2 border-gray-200 rounded-xl focus:border-red-500 focus:outline-none transition-colors text-lg"
                                required
                            />
                            <Youtube className="absolute right-4 top-1/2 transform -translate-y-1/2 w-6 h-6 text-gray-400" />
                        </div>

                        <button
                            onClick={handleSubmit}
                            disabled={loading || !url.trim()}
                            className="w-full bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white font-semibold py-4 px-8 rounded-xl transition-colors flex items-center justify-center gap-3 text-lg"
                        >
                            {loading ? (
                                <>
                                    <Loader2 className="w-6 h-6 animate-spin" />
                                    Analyzing Playlist...
                                </>
                            ) : (
                                <>
                                    <Clock className="w-6 h-6" />
                                    Calculate Duration
                                </>
                            )}
                        </button>
                    </div>

                    {error && (
                        <div className="max-w-2xl mx-auto mt-6 bg-red-50 border border-red-200 rounded-xl p-4 flex items-center gap-3">
                            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0" />
                            <p className="text-red-700">{error}</p>
                        </div>
                    )}
                </div>

                {/* Results Section */}
                {result && (
                    <div className="space-y-8">
                        {/* Summary Cards */}
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                            <div className="bg-white rounded-2xl shadow-lg border border-red-100 p-6">
                                <div className="flex items-center gap-4">
                                    <div className="bg-red-100 p-3 rounded-xl">
                                        <Clock className="w-6 h-6 text-red-600" />
                                    </div>
                                    <div>
                                        <p className="text-gray-600 text-sm">Total Duration</p>
                                        <p className="text-2xl font-bold text-gray-900">
                                            {formatDuration(result.total_duration_sec)}
                                        </p>
                                    </div>
                                </div>
                            </div>

                            <div className="bg-white rounded-2xl shadow-lg border border-red-100 p-6">
                                <div className="flex items-center gap-4">
                                    <div className="bg-blue-100 p-3 rounded-xl">
                                        <Film className="w-6 h-6 text-blue-600" />
                                    </div>
                                    <div>
                                        <p className="text-gray-600 text-sm">Total Videos</p>
                                        <p className="text-2xl font-bold text-gray-900">{result.videos.length}</p>
                                    </div>
                                </div>
                            </div>

                            <div className="bg-white rounded-2xl shadow-lg border border-red-100 p-6">
                                <div className="flex items-center gap-4">
                                    <div className="bg-green-100 p-3 rounded-xl">
                                        <Calendar className="w-6 h-6 text-green-600" />
                                    </div>
                                    <div>
                                        <p className="text-gray-600 text-sm">Average Duration</p>
                                        <p className="text-2xl font-bold text-gray-900">
                                            {formatDuration(Math.round(result.total_duration_sec / result.videos.length))}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>

                        {/* Playlist Info */}
                        <div className="bg-white rounded-2xl shadow-lg border border-red-100 overflow-hidden">
                            <div className="relative h-48 bg-gradient-to-r from-red-600 to-red-700">
                                <div className="absolute inset-0 bg-black bg-opacity-30"></div>
                                <img
                                    src={result.thumbnail}
                                    alt={result.title}
                                    className="w-full h-full object-cover mix-blend-overlay"
                                />
                                <div className="absolute bottom-6 left-6 right-6">
                                    <h3 className="text-2xl font-bold text-white mb-2">{result.title}</h3>
                                    <div className="flex items-center gap-2 text-red-100">
                                        <Users className="w-4 h-4" />
                                        <span className="text-sm">YouTube Playlist</span>
                                    </div>
                                </div>
                            </div>

                            {result.description && (
                                <div className="p-6">
                                    <p className="text-gray-700 leading-relaxed">
                                        {result.description.length > 300
                                            ? result.description.substring(0, 300) + "..."
                                            : result.description
                                        }
                                    </p>
                                </div>
                            )}
                        </div>

                        {/* Videos List */}
                        <div className="bg-white rounded-2xl shadow-lg border border-red-100">
                            <div className="p-6 border-b border-gray-100">
                                <h3 className="text-xl font-semibold text-gray-900 flex items-center gap-3">
                                    <Film className="w-6 h-6 text-red-600" />
                                    Videos in Playlist
                                </h3>
                            </div>

                            <div className="divide-y divide-gray-100 max-h-96 overflow-y-auto">
                                {result.videos.map((video, index) => (
                                    <div key={video.id} className="p-4 hover:bg-gray-50 transition-colors">
                                        <div className="flex gap-4">
                                            <a
                                                href={`https://www.youtube.com/watch?v=${video.id}`}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="flex-shrink-0 hover:opacity-80 transition-opacity"
                                            >
                                                <img
                                                    src={video.thumbnail}
                                                    alt={video.title}
                                                    className="w-20 h-14 object-cover rounded-lg"
                                                />
                                            </a>
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-start justify-between gap-4">
                                                    <div className="flex-1">
                                                        <a
                                                            href={`https://www.youtube.com/watch?v=${video.id}`}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="font-medium text-gray-900 line-clamp-2 leading-snug hover:text-red-600 transition-colors cursor-pointer"
                                                        >
                                                            {index + 1}. {video.title}
                                                        </a>
                                                        <div className="flex items-center gap-4 mt-2 text-sm text-gray-600">
                                                            <div className="flex items-center gap-1">
                                                                <Clock className="w-4 h-4" />
                                                                {formatDuration(video.duration_sec)}
                                                            </div>
                                                        </div>
                                                    </div>
                                                    <a
                                                        href={`https://www.youtube.com/watch?v=${video.id}`}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        className="text-red-600 hover:text-red-700 transition-colors"
                                                    >
                                                        <ExternalLink className="w-5 h-5" />
                                                    </a>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                )}
            </div>

            {/* Footer */}
            <footer className="py-8 border-t border-red-100 bg-white mt-auto">
                <div className="max-w-6xl mx-auto px-6">
                    <div className="flex items-center justify-center gap-3">
                        <a href="https://github.com/HaniAlKhaffaf" target="_blank" rel="noopener noreferrer">
                            <div className="bg-gray-900 p-2 rounded-lg">
                                <Github className="w-5 h-5 text-white" />
                            </div>
                        </a>
                        <a
                            href="https://github.com/HaniAlKhaffaf"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-900 font-semibold hover:text-red-600 transition-colors"
                        >
                            Hani Al Khaffaf
                        </a>
                    </div>
                </div>
            </footer>
        </div>
    );
};

export default PlaylistLengthCalculator;